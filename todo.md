# Implementation Todo List

## Phase 1: Foundation (API Types and Structure)

- [x] Add Cosign Dependency
  - [x] Update go.mod and go.sum files
  - [x] Test basic Cosign functionality

- [x] Create API Types
  - [x] Add ImageSignatures collector type in collector_shared.go
  - [x] Add ImageSignatures analyzer type in analyzer_shared.go
  - [x] Update Collect and Analyze structs to include new types

- [x] Add Registration Code
  - [x] Ensure types are properly registered
  - [x] Update deepcopy generation

## Phase 2: Collector Implementation

- [x] Basic Collector Structure
  - [x] Create pkg/collect/image_signatures.go
  - [x] Implement interface methods (Title, IsExcluded, Collect)

- [x] Registry Authentication
  - [x] Reuse registry.go auth methods
  - [x] Test connection to registries

- [x] Signature Retrieval
  - [x] Add methods to fetch signatures using Cosign
  - [x] Format and store signature data (JSON)

- [x] Error Handling
  - [x] Handle registry errors
  - [x] Handle missing signatures
  - [x] Support air-gapped environments

## Phase 3: Analyzer Implementation

- [x] Basic Analyzer Structure
  - [x] Create pkg/analyze/image_signatures.go
  - [x] Implement interface methods (Title, IsExcluded, Analyze)

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
