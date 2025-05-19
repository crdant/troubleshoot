# Implementation Todo List

## Phase 1: Foundation (API Types and Structure)

- [x] Add Cosign Dependency
  - [x] Update go.mod and go.sum files
  - [x] Test basic Cosign functionality

- [ ] Create API Types
  - [ ] Add ImageSignatures collector type in collector_shared.go
  - [ ] Add ImageSignatures analyzer type in analyzer_shared.go
  - [ ] Update Collect and Analyze structs to include new types

- [ ] Add Registration Code
  - [ ] Ensure types are properly registered
  - [ ] Update deepcopy generation

## Phase 2: Collector Implementation

- [ ] Basic Collector Structure
  - [ ] Create pkg/collect/image_signatures.go
  - [ ] Implement interface methods (Title, IsExcluded, Collect)

- [ ] Registry Authentication
  - [ ] Reuse registry.go auth methods
  - [ ] Test connection to registries

- [ ] Signature Retrieval
  - [ ] Add methods to fetch signatures using Cosign
  - [ ] Format and store signature data (JSON)

- [ ] Error Handling
  - [ ] Handle registry errors
  - [ ] Handle missing signatures
  - [ ] Support air-gapped environments

## Phase 3: Analyzer Implementation

- [ ] Basic Analyzer Structure
  - [ ] Create pkg/analyze/image_signatures.go
  - [ ] Implement interface methods (Title, IsExcluded, Analyze)

- [ ] Verification Logic
  - [ ] Add methods to verify signatures with Cosign
  - [ ] Support various verification methods (key, keyless, keyring)

- [ ] Outcome Generation
  - [ ] Implement condition matching
  - [ ] Generate appropriate outcomes based on conditions

## Phase 4: Testing and Integration

- [ ] Collector Unit Tests
  - [ ] Create test fixtures
  - [ ] Test basic collector functionality
  - [ ] Test with various registry types
  - [ ] Test error handling

- [ ] Analyzer Unit Tests
  - [ ] Create test fixtures
  - [ ] Test verification logic
  - [ ] Test condition matching
  - [ ] Test outcome generation

- [ ] Integration Tests
  - [ ] Test collector and analyzer together
  - [ ] Verify in local Kubernetes cluster

- [ ] Documentation
  - [ ] Add example YAML files
  - [ ] Update project documentation
  - [ ] Document usage patterns