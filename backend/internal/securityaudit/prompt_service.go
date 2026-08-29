package securityaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PromptService struct {
	config    ConfigStore
	repo      *PostgreSQLRepository
	payload   PayloadStore
	state     DeepReviewStateStore
	receipts  AllowReceiptStore
	enqueuer  *Enqueuer
	runner    *Runner
	evaluator *GuardEvaluator
	scanner   *OpenAICompatibleScanner
	metrics   *AtomicMetrics
	clock     Clock

	lifecycleMu  sync.Mutex
	cancel       context.CancelFunc
	background   context.Context
	enqueueWG    sync.WaitGroup
	enqueueSlots chan struct{}
	probeMu      sync.RWMutex
	probes       map[string]ProbeResult
}

func NewPromptService(
	config ConfigStore,
	repo *PostgreSQLRepository,
	payload PayloadStore,
	scanner *OpenAICompatibleScanner,
	metrics *AtomicMetrics,
) *PromptService {
	state, _ := payload.(DeepReviewStateStore)
	receipts, _ := payload.(AllowReceiptStore)
	var jobRepo JobRepository
	if repo != nil {
		jobRepo = repo
	}
	enqueuer := NewEnqueuer(config, jobRepo, payload, metrics)
	evaluator := NewGuardEvaluator(scanner, jobRepo, metrics)
	runner := NewRunner(config, jobRepo, payload, scanner, metrics)
	return &PromptService{
		config: config, repo: repo, payload: payload, state: state, receipts: receipts, scanner: scanner, metrics: metrics,
		enqueuer: enqueuer, evaluator: evaluator, runner: runner, clock: realClock{},
		enqueueSlots: make(chan struct{}, 128), probes: map[string]ProbeResult{},
	}
}

func (s *PromptService) Start(ctx context.Context) error {
	if s == nil || s.config == nil || s.runner == nil {
		logPromptRuntimeFailure(EventProcessFailed, "service_dependencies_unavailable")
		return errors.New("prompt audit service unavailable")
	}
	s.lifecycleMu.Lock()
	if s.cancel != nil {
		s.lifecycleMu.Unlock()
		return nil
	}
	background, cancel := context.WithCancel(ctx)
	s.background, s.cancel = background, cancel
	s.lifecycleMu.Unlock()
	configErr := s.config.Start(background)
	workerErr := s.runner.Start(background)
	if configErr != nil {
		logPromptRuntimeFailure(EventConfigReloadDegraded, "config_start_failed")
	}
	if workerErr != nil {
		logPromptRuntimeFailure(EventProcessFailed, "worker_start_failed")
	}
	return errors.Join(configErr, workerErr)
}

func (s *PromptService) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.lifecycleMu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
	}
	var workerErr error
	if s.runner != nil {
		workerErr = s.runner.Shutdown(ctx)
	}
	done := make(chan struct{})
	go func() { s.enqueueWG.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		if workerErr == nil {
			workerErr = ctx.Err()
		}
	}
	var configErr error
	if s.config != nil {
		configErr = s.config.Shutdown(ctx)
	}
	if workerErr != nil {
		logPromptRuntimeFailure(EventProcessFailed, "worker_shutdown_failed")
		return workerErr
	}
	if configErr != nil {
		logPromptRuntimeFailure(EventConfigReloadDegraded, "config_shutdown_failed")
	}
	return configErr
}

func (s *PromptService) EffectiveMode() Mode {
	if s == nil || s.config == nil {
		return ModeOff
	}
	return s.config.EffectiveMode()
}

func (s *PromptService) BlockingApplies(req Request) bool {
	if s == nil || s.config == nil || s.config.BlockingActivationDegraded() {
		return false
	}
	cfg, ok := s.config.Active()
	return ok && cfg.EffectiveMode() == ModeBlocking && cfg.IncludesGroup(req.GroupID)
}

func (s *PromptService) Enqueue(_ context.Context, req Request) error {
	return s.enqueue(req, ModeAsync)
}

func (s *PromptService) EnqueueDeep(_ context.Context, req Request) error {
	return s.enqueue(req, ModeAsyncDeep)
}

func (s *PromptService) enqueue(req Request, mode Mode) error {
	expectedMode := ModeAsync
	if mode == ModeAsyncDeep {
		expectedMode = ModeBlocking
	}
	if s == nil || s.EffectiveMode() != expectedMode {
		return nil
	}
	if s.enqueuer == nil {
		if s.metrics != nil {
			s.metrics.IncDropped()
		}
		LogWarn(EventEnqueueDropped, mergeLogFields(requestLogFields(req), map[string]any{"status": "dropped", "error_code": "enqueuer_unavailable", "error_kind": "audit_dependency"}))
		return nil
	}
	select {
	case s.enqueueSlots <- struct{}{}:
	default:
		if s.metrics != nil {
			s.metrics.IncDropped()
		}
		LogWarn(EventEnqueueDropped, mergeLogFields(requestLogFields(req), map[string]any{"status": "dropped", "error_code": "local_enqueue_busy", "error_kind": "audit_dependency"}))
		return nil
	}
	s.lifecycleMu.Lock()
	background := s.background
	s.lifecycleMu.Unlock()
	if background == nil {
		<-s.enqueueSlots
		LogWarn(EventEnqueueDropped, mergeLogFields(requestLogFields(req), map[string]any{"status": "dropped", "error_code": "service_not_started", "error_kind": "audit_dependency"}))
		return errors.New("prompt audit service not started")
	}
	requestCopy := req.Clone()
	s.enqueueWG.Add(1)
	go func() {
		defer s.enqueueWG.Done()
		defer func() { <-s.enqueueSlots }()
		ctx, cancel := context.WithTimeout(background, 2*time.Second)
		defer cancel()
		if mode == ModeAsyncDeep {
			_ = s.enqueuer.EnqueueDeep(ctx, requestCopy)
		} else {
			_ = s.enqueuer.Enqueue(ctx, requestCopy)
		}
	}()
	return nil
}

func (s *PromptService) Evaluate(ctx context.Context, req Request) (*PromptDecision, error) {
	if s == nil || s.config == nil || s.evaluator == nil {
		logPromptRequestFailure(req, DecisionUnavailable, ErrorCodeUnavailable)
		return nil, &GuardError{Code: ErrorCodeUnavailable}
	}
	if s.config.BlockingActivationDegraded() {
		logPromptRequestFailure(req, DecisionUnavailable, ErrorCodeUnavailable)
		return nil, &GuardError{Code: ErrorCodeUnavailable}
	}
	cfg, ok := s.config.Active()
	if !ok {
		if s.config.EffectiveMode() == ModeBlocking {
			logPromptRequestFailure(req, DecisionUnavailable, ErrorCodeUnavailable)
			return nil, &GuardError{Code: ErrorCodeUnavailable}
		}
		return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true}, nil
	}
	if cfg.EffectiveMode() != ModeBlocking {
		return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true}, nil
	}
	stateToken, deepRequired, stateErr := s.deepReviewRequired(ctx, req.UserID)
	if stateErr != nil {
		logPromptRequestFailure(req, DecisionUnavailable, ErrorCodeDeepReviewState)
		return nil, &GuardError{Code: ErrorCodeDeepReviewState, Cause: stateErr}
	}
	if !deepRequired && !cfg.IncludesGroup(req.GroupID) {
		return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true}, nil
	}
	if deepRequired {
		stateToken, stateErr = s.claimRequiredDeepReview(ctx, req, stateToken)
		if stateErr != nil {
			logPromptRequestFailure(req, DecisionUnavailable, ErrorCodeDeepReviewState)
			return nil, &GuardError{Code: ErrorCodeDeepReviewState, Cause: stateErr}
		}
	}
	var snapshot PromptSnapshot
	var diagnostic promptExtractionDiagnostic
	var err error
	if deepRequired {
		snapshot, diagnostic, err = extractDeepPromptSnapshotWithDiagnostics(req, cfg.DeepReviewModules)
	} else {
		snapshot, diagnostic, err = extractBlockingPromptSnapshotWithDiagnostics(req, cfg.BlockingReviewModules)
	}
	if diagnostic.Failed {
		if s.metrics != nil {
			s.metrics.ObserveExtraction(ExtractionFailed)
		}
		logPromptExtractionFailure(req, diagnostic)
	}
	if diagnostic.Failed {
		return nil, &GuardError{Code: ErrorCodeExtractionFailed, Cause: err}
	}
	if errors.Is(err, ErrNoPromptText) {
		if s.metrics != nil {
			s.metrics.ObserveExtraction(ExtractionEmpty)
		}
		if deepRequired {
			return &PromptDecision{Kind: DecisionBlock, ErrorCode: ErrorCodeDeepReviewRequired, AllowNextStage: false, DeepReviewed: true}, nil
		}
		return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true, DeepReviewed: deepRequired}, nil
	}
	if err != nil {
		if !diagnostic.Failed {
			diagnostic = promptExtractionDiagnostic{Failed: true, ErrorCode: "content_extraction_failed"}
			if s.metrics != nil {
				s.metrics.ObserveExtraction(ExtractionFailed)
			}
			logPromptExtractionFailure(req, diagnostic)
		}
		return nil, &GuardError{Code: ErrorCodeExtractionFailed, Cause: err}
	}
	prepareAllowReceipts(ctx, s.receipts, s.metrics, cfg, &snapshot, req.AllowReceiptKeys, deepRequired)
	if strings.TrimSpace(snapshot.ScanText) == "" {
		if s.metrics != nil {
			s.metrics.ObserveExtraction(ExtractionSucceeded)
		}
		if deepRequired {
			return &PromptDecision{Kind: DecisionBlock, ErrorCode: ErrorCodeDeepReviewRequired, AllowNextStage: false, DeepReviewed: true}, nil
		}
		return &PromptDecision{Kind: DecisionAllow, AllowNextStage: true}, nil
	}
	ciphertext, err := encryptCompletePromptContext(s.config, snapshot.CompleteContext)
	if err != nil {
		logPromptRequestFailure(req, DecisionUnavailable, ErrorCodeEncryptionKeyRequired)
		return nil, &GuardError{Code: ErrorCodeEncryptionKeyRequired, Cause: err}
	}
	snapshot.FullContextCiphertext = ciphertext
	snapshot.CompleteContext = ""
	if s.metrics != nil {
		s.metrics.ObserveExtraction(ExtractionSucceeded)
	}
	decision, err := s.evaluator.Evaluate(ctx, cfg, snapshot)
	if err == nil && decision != nil && decision.Kind == DecisionAllow {
		decision.AllowReceiptKeys = append([]string(nil), snapshot.AllowReceiptKeys...)
		decision.allowReceipt = &allowReceiptCommit{ConfigVersion: cfg.ConfigVersion, Snapshot: snapshot}
	}
	if err != nil || !deepRequired {
		return decision, err
	}
	return s.finishRequiredDeepReview(ctx, req, stateToken, decision)
}

func (s *PromptService) commitAllowReceipts(ctx context.Context, decision *PromptDecision) {
	if s == nil || decision == nil || decision.Kind != DecisionAllow || decision.allowReceipt == nil {
		return
	}
	commit := decision.allowReceipt
	decision.allowReceipt = nil
	cfg, ok := s.config.Active()
	if !ok || cfg.ConfigVersion != commit.ConfigVersion {
		logAllowReceiptFailure(commit.Snapshot, "store", "allow_receipt_config_changed")
		return
	}
	storeAllowReceipts(ctx, s.receipts, s.metrics, cfg, commit.Snapshot)
}

func (s *PromptService) deepReviewRequired(ctx context.Context, userID int64) (string, bool, error) {
	if userID <= 0 {
		return "", false, nil
	}
	if s == nil || s.state == nil {
		return "", false, errors.New("prompt audit deep review state unavailable")
	}
	return s.state.Required(ctx, userID)
}

func (s *PromptService) claimRequiredDeepReview(ctx context.Context, req Request, token string) (string, error) {
	claim := fmt.Sprintf("review:%s:%d", req.RequestID, s.clock.Now().UnixNano())
	replaced, err := s.state.Replace(ctx, req.UserID, token, claim)
	if err != nil {
		return "", err
	}
	if !replaced {
		return "", errors.New("prompt audit deep review state changed")
	}
	return claim, nil
}

func (s *PromptService) finishRequiredDeepReview(ctx context.Context, req Request, stateToken string, decision *PromptDecision) (*PromptDecision, error) {
	if decision == nil || s == nil || s.state == nil {
		return nil, &GuardError{Code: ErrorCodeDeepReviewState}
	}
	decision.DeepReviewed = true
	if decision.Kind == DecisionAllow {
		cleared, err := s.state.Clear(ctx, req.UserID, stateToken)
		if err != nil || !cleared {
			if err == nil {
				err = errors.New("prompt audit deep review state changed")
			}
			return nil, &GuardError{Code: ErrorCodeDeepReviewState, Cause: err}
		}
		return decision, nil
	}
	if decision.Kind == DecisionFlag || decision.Kind == DecisionBlock {
		decision.Kind = DecisionBlock
		decision.ErrorCode = ErrorCodeDeepReviewRequired
		decision.AllowNextStage = false
	}
	return decision, nil
}

func (s *PromptService) GetConfig() (PublicConfig, error) { return s.config.Public() }

func (s *PromptService) SaveConfig(ctx context.Context, req UpdateConfigRequest, actorID int64) (PublicConfig, error) {
	return s.config.Save(ctx, req, actorID)
}

func (s *PromptService) Runtime(ctx context.Context) RuntimeSnapshot {
	expected, activeVersion, loadedAt, loadError := s.config.RuntimeState()
	cfg, hasConfig := s.config.Active()
	mode := s.EffectiveMode()
	workerTotal, queueCapacity := 0, 0
	if hasConfig {
		workerTotal, queueCapacity = cfg.WorkerCount, cfg.QueueCapacity
	}
	runtime := RuntimeSnapshot{
		ProcessStatus: "disabled", EffectiveMode: mode, ExpectedConfigVersion: expected,
		ActiveConfigVersion: activeVersion, ConfigLoadedAt: loadedAt, ConfigLoadError: loadError,
		WorkerTotal: workerTotal, QueueCapacity: queueCapacity, DatabaseStatus: "ok", RedisStatus: "ok",
		Endpoints: s.probeSnapshot(), GuardMetrics: s.metrics.Snapshot(),
	}
	if s.repo != nil {
		stats, err := s.repo.QueueStats(ctx)
		if err != nil {
			runtime.DatabaseStatus = "error"
			runtime.LastErrorCode = "database_unavailable"
			now := s.clock.Now()
			runtime.LastErrorAt = &now
			logPromptRuntimeFailure(EventProcessFailed, "queue_stats_failed")
		} else {
			runtime.Queue = stats
		}
	} else {
		runtime.DatabaseStatus = "error"
	}
	payloadUnavailable := s.payload == nil
	if !payloadUnavailable {
		payloadUnavailable = s.payload.Ping(ctx) != nil
	}
	if payloadUnavailable {
		runtime.RedisStatus = "error"
		if runtime.LastErrorCode == "" {
			runtime.LastErrorCode = "payload_store_unavailable"
			now := s.clock.Now()
			runtime.LastErrorAt = &now
		}
		logPromptRuntimeFailure(EventProcessFailed, "payload_store_unavailable")
	}
	activeWorkers, processed, failed, heartbeat, lastProcessed, workerCode, workerMessage := s.runner.Snapshot()
	runtime.WorkerActive, runtime.ProcessedTotal, runtime.FailedTotal = activeWorkers, processed, failed
	if s.metrics != nil {
		auditMetrics := s.metrics.AuditSnapshot()
		runtime.EnqueuedTotal, runtime.DroppedTotal = auditMetrics.Enqueued, auditMetrics.Dropped
		runtime.ExtractionAttempted = auditMetrics.ExtractionAttempted
		runtime.ExtractionSucceeded = auditMetrics.ExtractionSucceeded
		runtime.ExtractionEmpty = auditMetrics.ExtractionEmpty
		runtime.ExtractionFailed = auditMetrics.ExtractionFailed
		runtime.AllowReceiptHits = auditMetrics.AllowReceiptHits
		runtime.AllowReceiptMisses = auditMetrics.AllowReceiptMisses
		runtime.AllowReceiptWrites = auditMetrics.AllowReceiptWrites
		runtime.AllowReceiptErrors = auditMetrics.AllowReceiptErrors
	}
	runtime.WorkerHeartbeatAt, runtime.LastProcessedAt = heartbeat, lastProcessed
	if workerCode != "" {
		runtime.LastErrorCode, runtime.LastErrorMessage = workerCode, workerMessage
		runtime.LastErrorAt = s.runner.LastErrorAt()
	}
	if mode != ModeOff {
		runtime.ProcessStatus = "running"
		if loadError != "" || runtime.DatabaseStatus != "ok" || runtime.RedisStatus != "ok" || activeVersion != expected {
			runtime.ProcessStatus = "degraded"
		}
		if heartbeat == nil || s.clock.Now().Sub(*heartbeat) > 10*time.Second {
			runtime.ProcessStatus = "degraded"
		}
	}
	return runtime
}

type ProbeRequest struct {
	Endpoint UpdateEndpoint `json:"endpoint"`
}

func (s *PromptService) Probe(ctx context.Context, request ProbeRequest) ProbeResult {
	started := s.clock.Now()
	endpoint, tokenApplied, err := s.resolveProbeEndpoint(request.Endpoint)
	if err != nil {
		return s.finishProbe(request.Endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: "endpoint_invalid", Message: "审计节点配置无效"})
	}
	LogInfo(EventProbeStarted, map[string]any{"guard_endpoint_id": endpoint.ID, "status": "started"})
	client, err := NewSecureHTTPClient(endpoint)
	if err != nil {
		return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: "endpoint_unsafe", Message: "审计节点地址不在允许范围", TokenApplied: tokenApplied})
	}
	modelsURL, _ := ModelsURL(endpoint.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: "probe_request_invalid", Message: "无法创建探测请求", TokenApplied: tokenApplied})
	}
	if endpoint.Token != "" {
		req.Header.Set("Authorization", "Bearer "+endpoint.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		code := "connection_failed"
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			code = "timeout"
		}
		return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: code, Message: "无法连接审计节点", Retryable: true, TokenApplied: tokenApplied})
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxGuardResponseBytes+1))
	_ = resp.Body.Close()
	if readErr != nil {
		return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: "response_read_failed", Message: "审计节点响应读取失败", HTTPStatus: resp.StatusCode, Retryable: true, TokenApplied: tokenApplied})
	}
	if int64(len(responseBody)) > maxGuardResponseBytes {
		return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: "response_too_large", Message: "审计节点响应无效", HTTPStatus: resp.StatusCode, TokenApplied: tokenApplied})
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && modelsResponseReady(responseBody, endpoint.Model) {
		return s.finishProbe(endpoint.ID, started, ProbeResult{OK: true, Status: "healthy", Message: "审计节点连接正常", HTTPStatus: resp.StatusCode, TokenApplied: tokenApplied})
	}
	if (resp.StatusCode >= 200 && resp.StatusCode < 300) || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		result, scanErr := s.scanner.Scan(ctx, endpoint, "Hello", AllScannerIDs)
		if scanErr == nil && result != nil {
			return s.finishProbe(endpoint.ID, started, ProbeResult{OK: true, Status: "healthy", Message: "审计节点模型调用正常", HTTPStatus: http.StatusOK, TokenApplied: tokenApplied})
		}
		code, status, retryable := guardErrorCode(scanErr), 0, false
		var guardErr *GuardError
		if errors.As(scanErr, &guardErr) {
			status, retryable = guardErr.HTTPStatus, guardErr.Retryable
		}
		if code == "" {
			code = ErrorCodeInvalidResponse
		}
		return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: code, Message: "审计节点模型调用失败", HTTPStatus: status, Retryable: retryable, TokenApplied: tokenApplied})
	}
	code, retryable := "probe_http_error", resp.StatusCode == 429 || resp.StatusCode >= 500
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		code = "authentication_failed"
	}
	return s.finishProbe(endpoint.ID, started, ProbeResult{Status: "failed", ErrorCode: code, Message: "审计节点探测失败", HTTPStatus: resp.StatusCode, Retryable: retryable, TokenApplied: tokenApplied})
}

func modelsResponseReady(body []byte, model string) bool {
	var response struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &response) != nil || response.Data == nil {
		return false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return true
	}
	for _, item := range response.Data {
		if strings.TrimSpace(item.ID) == model {
			return true
		}
	}
	return false
}

func (s *PromptService) resolveProbeEndpoint(input UpdateEndpoint) (ActiveEndpoint, bool, error) {
	baseURL, err := NormalizeBaseURL(input.BaseURL)
	if err != nil {
		return ActiveEndpoint{}, false, err
	}
	token := strings.TrimSpace(input.Token)
	if token == "" {
		if cfg, ok := s.config.Active(); ok {
			for _, endpoint := range cfg.Endpoints {
				if endpoint.ID != strings.TrimSpace(input.ID) {
					continue
				}
				// Reuse a stored credential only when the probe targets the same
				// normalized base URL. Otherwise an admin probe could exfiltrate
				// the Guard token to an attacker-controlled HTTPS host.
				if endpoint.BaseURL == baseURL {
					token = endpoint.Token
				}
				break
			}
		}
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		model = DefaultGuardModel
	}
	timeout := input.TimeoutMS
	if timeout == 0 {
		timeout = DefaultTimeoutMS
	}
	limit := input.InputLimit
	if limit == 0 {
		limit = DefaultInputLimit
	}
	storage := storageConfig{Enabled: false, Strategy: "priority", WorkerCount: DefaultWorkerCount, QueueCapacity: DefaultQueueCapacity, AllowReceiptTTLSeconds: DefaultAllowReceiptTTLSeconds, Scanners: append([]string(nil), AllScannerIDs...), AllGroups: true,
		Endpoints: []StorageEndpoint{{ID: strings.TrimSpace(input.ID), Name: strings.TrimSpace(input.Name), Protocol: "openai_compatible", BaseURL: baseURL, Model: model, TimeoutMS: timeout, InputLimit: limit}}}
	if storage.Endpoints[0].ID == "" {
		storage.Endpoints[0].ID = "probe"
	}
	if storage.Endpoints[0].Name == "" {
		storage.Endpoints[0].Name = "Probe"
	}
	if err := validateStorageConfig(storage); err != nil {
		return ActiveEndpoint{}, false, err
	}
	return ActiveEndpoint{ID: storage.Endpoints[0].ID, Name: storage.Endpoints[0].Name, Protocol: "openai_compatible", BaseURL: baseURL, Model: model, Token: token, TimeoutMS: timeout, InputLimit: limit, Enabled: true}, token != "", nil
}

func (s *PromptService) finishProbe(id string, started time.Time, result ProbeResult) ProbeResult {
	result.CheckedAt = s.clock.Now()
	result.LatencyMS = int(result.CheckedAt.Sub(started).Milliseconds())
	if result.OK {
		LogInfo(EventProbeFinished, map[string]any{"guard_endpoint_id": id, "status": result.Status, "latency_ms": result.LatencyMS, "http_status": result.HTTPStatus})
	} else {
		LogWarn(EventProbeFailed, map[string]any{"guard_endpoint_id": id, "status": result.Status, "latency_ms": result.LatencyMS, "http_status": result.HTTPStatus, "error_code": result.ErrorCode, "retryable": result.Retryable})
	}
	s.probeMu.Lock()
	s.probes[id] = result
	s.probeMu.Unlock()
	return result
}

func (s *PromptService) probeSnapshot() map[string]ProbeResult {
	s.probeMu.RLock()
	defer s.probeMu.RUnlock()
	result := make(map[string]ProbeResult, len(s.probes))
	for id, probe := range s.probes {
		result[id] = probe
	}
	return result
}

func (s *PromptService) ListEvents(ctx context.Context, filter EventFilter, page, pageSize int) (*EventPage, error) {
	return s.repo.ListEvents(ctx, filter, page, pageSize)
}
func (s *PromptService) GetEvent(ctx context.Context, id int64) (*Event, error) {
	return s.repo.GetEvent(ctx, id)
}

func (s *PromptService) DownloadEventContext(ctx context.Context, id int64) (*EventContextDownload, error) {
	record, err := s.repo.GetEventContext(ctx, id)
	if err != nil {
		return nil, err
	}
	raw, err := decryptCompletePromptContext(s.config, record.Ciphertext)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != record.SHA256 || !json.Valid(raw) {
		return nil, errors.New("prompt audit event context integrity check failed")
	}
	return &EventContextDownload{JSON: raw, SHA256: record.SHA256}, nil
}

func (s *PromptService) DeleteEvent(ctx context.Context, id int64) (*DeleteResult, error) {
	result, err := s.repo.DeleteEvent(ctx, id)
	if err == nil {
		s.deletePayloads(ctx, result.JobIDs)
	}
	return result, err
}
func (s *PromptService) DeleteEventsByIDs(ctx context.Context, ids []int64) (*DeleteResult, error) {
	result, err := s.repo.DeleteEventsByIDs(ctx, ids)
	if err == nil {
		s.deletePayloads(ctx, result.JobIDs)
	}
	return result, err
}

type deleteClaims struct {
	FilterHash    string    `json:"filter_hash"`
	SnapshotMaxID int64     `json:"snapshot_max_id"`
	AdminID       int64     `json:"admin_id"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func (s *PromptService) PreviewDelete(ctx context.Context, filter EventFilter, adminID int64) (*DeletePreview, error) {
	preview, err := s.repo.PreviewDelete(ctx, filter)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	expires := now.Add(5 * time.Minute)
	claimsRaw, _ := json.Marshal(deleteClaims{FilterHash: preview.FilterHash, SnapshotMaxID: preview.SnapshotMaxID, AdminID: adminID, IssuedAt: now, ExpiresAt: expires})
	token, err := s.config.Encrypt(string(claimsRaw))
	if err != nil {
		return nil, err
	}
	preview.ConfirmationToken, preview.ExpiresAt = token, expires
	LogInfo(EventDeletePreviewed, map[string]any{"user_id": adminID, "status": "previewed"})
	return preview, nil
}

type DeleteByFilterRequest struct {
	Filter            EventFilter `json:"filter"`
	SnapshotMaxID     int64       `json:"snapshot_max_id"`
	FilterHash        string      `json:"filter_hash"`
	ConfirmationToken string      `json:"confirmation_token"`
	Confirm           bool        `json:"confirm"`
}

func (s *PromptService) DeleteByFilter(ctx context.Context, request DeleteByFilterRequest, adminID int64) (*DeleteResult, error) {
	if !request.Confirm {
		return nil, errors.New("prompt audit filter delete requires confirm=true")
	}
	plain, err := s.config.Decrypt(strings.TrimSpace(request.ConfirmationToken))
	if err != nil {
		return nil, errors.New("prompt audit confirmation token invalid")
	}
	var claims deleteClaims
	if json.Unmarshal([]byte(plain), &claims) != nil {
		return nil, errors.New("prompt audit confirmation token invalid")
	}
	computed := FilterHash(request.Filter, request.SnapshotMaxID)
	if claims.AdminID != adminID || claims.SnapshotMaxID != request.SnapshotMaxID || claims.FilterHash != request.FilterHash || request.FilterHash != computed || !s.clock.Now().Before(claims.ExpiresAt) {
		return nil, errors.New("prompt audit confirmation token does not match deletion request")
	}
	result, err := s.repo.DeleteEventsByFilter(ctx, request.Filter, request.SnapshotMaxID, 200)
	if err == nil {
		s.deletePayloads(ctx, result.JobIDs)
		LogWarn(EventEventsFilterDeleted, map[string]any{"user_id": adminID, "status": "deleted"})
	}
	return result, err
}

func (s *PromptService) deletePayloads(ctx context.Context, jobIDs []int64) {
	for _, id := range jobIDs {
		_ = s.payload.Delete(ctx, id)
	}
}

func parseTimeQuery(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}
