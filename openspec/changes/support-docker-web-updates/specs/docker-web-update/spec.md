## ADDED Requirements

### Requirement: Docker website updates must use a persistent executable

Docker deployments SHALL run the application from an executable stored under
the existing persistent data mount. The image-provided executable MUST remain an
immutable bootstrap source, and the application process MUST remain non-root and
MUST NOT receive access to the Docker socket.

#### Scenario: Administrator installs an update from the website

- **WHEN** the authenticated update endpoint downloads and verifies a compatible
  release in a Docker deployment
- **THEN** it SHALL atomically replace the executable in the persistent runtime
  directory
- **THEN** the current process SHALL continue serving until the administrator
  invokes the existing restart action
- **THEN** the restarted container SHALL execute the installed release

#### Scenario: The same container image is recreated

- **WHEN** Docker recreates a container with the same image build identity and
  both the persistent data and trusted image-state mounts
- **THEN** the entrypoint SHALL preserve and execute the website-installed
  runtime binary

### Requirement: Explicit image selection must remain authoritative

The entrypoint SHALL associate the persistent runtime binary with the image build
that seeded it using a separate root-only persistent marker outside the
application-writable data mount. A different image build identity MUST replace
the persistent runtime binary with that image's executable before application
startup.

#### Scenario: Operator changes or rolls back the image

- **WHEN** the selected image build identity differs from the recorded identity
- **THEN** the image executable SHALL atomically replace the persistent runtime
  executable
- **THEN** an obsolete local binary backup SHALL be removed
- **THEN** the container SHALL execute the explicitly selected image version

#### Scenario: Runtime seeding was interrupted

- **WHEN** the runtime executable is missing, non-executable, or associated with
  a different image build
- **THEN** the next startup SHALL first restore a complete regular executable
  backup associated with the same image build when one exists
- **THEN** it SHALL otherwise recover from the immutable image executable

### Requirement: Docker website updates must retain existing controls

Docker website updates SHALL continue to use the existing administrator
authorization, trusted download hosts, archive bounds, checksum verification,
operation lock, atomic replacement, and explicit restart behavior.

#### Scenario: Application is compromised or an update is invalid

- **WHEN** an update does not pass the existing download or checksum controls
- **THEN** it MUST NOT replace the persistent runtime executable
- **THEN** the application process MUST NOT be able to read, replace, or forge
  the trusted selected-image marker
- **THEN** the application MUST NOT gain Docker daemon or host orchestration
  privileges as part of the update design
