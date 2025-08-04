# Chaincode Upgrade: Brainstorming and Development Plan

This document outlines the plan for upgrading the `documentApproval` chaincode with new features to create a more robust and flexible system.

## 1. Brainstorming & Feature Analysis

This section details the new features and analyzes the best architectural approach for implementation (Chaincode vs. Application Layer).

### Guiding Principle: Chaincode vs. Application

*   **Chaincode (Smart Contract):** The chaincode is the ultimate source of truth. Any rule that is **non-negotiable** and must be enforced for the integrity of the system belongs here. Think of it as the "law" of your application. If it's in the chaincode, no one can cheat the system, not even a buggy frontend.
*   **Application (Frontend/Gateway):** The application layer is responsible for user experience, driving processes, and interacting with the outside world (like sending notifications). It reads data from the chaincode and presents it to the user, and it initiates transactions based on user actions.

### Feature Breakdown

#### 1. Approver Comments
*   **Description:** When an approver makes a decision, they can attach a text comment explaining their reasoning.
*   **Implementation:** **Chaincode**. The comment is part of the official, auditable decision and must be stored immutably on the ledger.

#### 2. Customizable Decision Options
*   **Description:** The document submitter can define a custom set of possible outcomes for each document (e.g., ["ACCEPT", "ACCEPT_WITH_REVISIONS", "REJECT"]).
*   **Implementation:** **Chaincode**. The set of valid decisions is a core rule for that specific document and must be enforced by the chaincode.

#### 3. Multi-Stage Workflows
*   **Description:** A document moves through a predefined series of approval stages. Rejection can send it back to a previous stage for revision.
*   **Implementation:** **Hybrid Approach**.
    *   **Chaincode:** The *rules* of the workflow (the definition of each stage, required approvers, transition logic) must be defined and enforced in the chaincode.
    *   **Application:** The application is responsible for *driving* the workflow by calling chaincode functions to advance the stage.

#### 4. Time-Based Deadlines & Notifications
*   **Description:** Documents can have a deadline for approval. The system can send reminders or take action if the deadline passes.
*   **Implementation:** **Hybrid Approach**.
    *   **Chaincode:** The `Deadline` (as a Unix timestamp) is a fact about the document and must be stored on the ledger.
    *   **Application:** An external service (automation bot) is required to query the chaincode, send notifications (email/SMS), and submit transactions to expire documents.

#### 5. Document Versioning & History
*   **Description:** When a document is rejected and resubmitted, a new version is created, providing a full, auditable history.
*   **Implementation:** **Chaincode**. The history of a document is a critical part of its lifecycle and must be stored on-chain.

---

## 2. Chaincode Upgrade Plan

This is a four-phase plan to implement the features incrementally.

### **Phase 1: Enhancing Decision-Making**

**Goal:** Implement the most direct improvements: allow approvers to add comments and allow submitters to define custom decision options.

**Breakdown:**

1.  **Modify Chaincode Structs:**
    *   In `chaincode.go`, update the `Decision` struct to include a `Comment` field.
    *   Update the `Document` struct to include a slice for `ValidDecisions`.

2.  **Update Chaincode Functions:**
    *   **`SubmitDocument`:** Modify to accept `validDecisionsJSON` and save it to the `Document` struct.
    *   **`ApproveDocument`:** Modify to accept a `comment`. It must validate the decision against the `ValidDecisions` slice.
    *   **`QueryDocumentStatus`:** Update to return the new `comment` and `validDecisions` fields.

3.  **Testing Plan for Phase 1:**
    *   Deploy the upgraded chaincode.
    *   Submit a document with custom decisions.
    *   Attempt an invalid decision; expect failure.
    *   Approve with a valid decision and a comment; expect success.
    *   Query the status and verify the new data is stored correctly.

---

### **Phase 2: Introducing Document Versioning**

**Goal:** Create a full, auditable history of a document as it is revised and resubmitted.

**Breakdown:**

1.  **Modify Chaincode Structs:**
    *   Create a new `DocumentVersion` struct: `{ Version int, Hash string, Submitter string, Timestamp int64 }`.
    *   Update the `Document` struct: Replace the single `Hash` field with a `Versions []DocumentVersion` slice and add `LatestVersion int`.

2.  **Update Chaincode Functions:**
    *   **`SubmitDocument`:** Logic changes to handle both new documents and new versions of existing documents by appending to the `Versions` slice.
    *   **`QueryDocumentStatus`:** Update to return the full `Versions` history.

3.  **Testing Plan for Phase 2:**
    *   Submit a document for the first time. Verify `LatestVersion` is 1.
    *   Submit the same document ID with a new hash. Verify `LatestVersion` is 2 and the `Versions` slice has two entries.

---

### **Phase 3: Implementing Multi-Stage Workflows**

**Goal:** Transform the single-approval process into a configurable, multi-stage workflow.

**Breakdown:**

1.  **Modify Chaincode Structs:**
    *   Create a new `WorkflowStage` struct: `{ StageName string, Approvers []string, RequiredApprovals int }`.
    *   Update `Document` struct to include `Workflow []WorkflowStage`, `CurrentStage int`, and `Status string`.

2.  **Update Chaincode Functions:**
    *   **`SubmitDocument`:** Modify to accept a `workflowJSON` argument.
    *   **`ApproveDocument`:** Becomes the workflow engine. Checks if the current stage's requirements are met and advances the `CurrentStage` or sets the `Status` to "COMPLETED".
    *   **Create `RejectDocument` function:** Explicitly sets the `Status` to "REJECTED" and can move the `CurrentStage` back for rework.

3.  **Testing Plan for Phase 3:**
    *   Submit a document with a two-stage workflow.
    *   Approve with one user; verify stage does not advance.
    *   Approve with a second user; verify `CurrentStage` is now 2.
    *   Reject from Stage 2; verify `Status` is "REJECTED".

---

### **Phase 4: Adding Deadlines & Automation Hooks**

**Goal:** Add time-based constraints and create functions that can be called by an external automation service.

**Breakdown:**

1.  **Modify Chaincode Structs:**
    *   Update the `Document` struct to include a `Deadline int64` (Unix timestamp).

2.  **Update Chaincode Functions:**
    *   **`SubmitDocument`:** Add a `deadline` argument.
    *   **Create `CheckDocumentStatus` function:** A read-only function for an external bot to find documents nearing their deadline.
    *   **Create `ExpireDocument` function:** A transaction function for a trusted bot to call to mark a document as "EXPIRED" if the deadline has passed.
	
3.  **Testing Plan for Phase 4:**
    *   Submit a document with a future deadline.
    *   Call `ExpireDocument` immediately; expect failure.
    *   Simulate time passing and call `ExpireDocument` again; expect success.