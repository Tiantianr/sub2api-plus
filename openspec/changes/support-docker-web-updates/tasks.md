## 1. Runtime contract

- [x] 1.1 Specify persistent Docker executable ownership and image precedence.
- [x] 1.2 Specify the existing update security and restart boundaries.

## 2. Container implementation

- [x] 2.1 Embed an immutable image build identity beside the bootstrap binary.
- [x] 2.2 Seed and run the application from the persistent data mount.
- [x] 2.3 Reset the runtime binary when an explicitly selected image changes.

## 3. Verification and documentation

- [x] 3.1 Add static deployment assertions and a real Linux image smoke test.
- [x] 3.2 Document website update persistence, image precedence, and bootstrap.
- [x] 3.3 Run deployment checks and strict OpenSpec validation.
