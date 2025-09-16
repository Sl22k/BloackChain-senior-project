package chaincode

import (
	"fmt"
	"sort"
	"strings"
)

// SortOrder represents different sorting options for documents
type SortOrder string

const (
	SortNewest SortOrder = "newest"
	SortOldest SortOrder = "oldest"
	SortTitle  SortOrder = "title"
)

// sortDocuments sorts a slice of documents based on the specified sort order
func sortDocuments(docs []*Document, sortOrder SortOrder) {
	switch sortOrder {
	case SortNewest:
		sort.Slice(docs, func(i, j int) bool {
			return docs[i].CreatedTimestamp > docs[j].CreatedTimestamp
		})
	case SortOldest:
		sort.Slice(docs, func(i, j int) bool {
			return docs[i].CreatedTimestamp < docs[j].CreatedTimestamp
		})
	case SortTitle:
		sort.Slice(docs, func(i, j int) bool {
			return docs[i].Title < docs[j].Title
		})
	default:
		// Default to newest
		sort.Slice(docs, func(i, j int) bool {
			return docs[i].CreatedTimestamp > docs[j].CreatedTimestamp
		})
	}
}

// parseSortOrder converts string to SortOrder enum
func parseSortOrder(sortOrderStr string) SortOrder {
	switch strings.ToLower(sortOrderStr) {
	case "newest":
		return SortNewest
	case "oldest":
		return SortOldest
	case "title":
		return SortTitle
	default:
		return SortNewest
	}
}

// limitDocuments returns a limited slice of documents
func limitDocuments(docs []*Document, limit int) []*Document {
	if len(docs) <= limit {
		return docs
	}
	return docs[:limit]
}

// calculateStatusCounts counts decisions by status for a document
func calculateStatusCounts(approvalsMap map[string]Decision) map[string]int {
	statusCounts := make(map[string]int)
	for _, decision := range approvalsMap {
		statusCounts[string(decision.Status)]++
	}
	return statusCounts
}

// determineDocumentStatus determines document status based on decision counts
func (s *SmartContract) determineDocumentStatus(decisionCounts map[string]int, validDecisions []string) string {
	if len(decisionCounts) == 0 {
		return "PENDING"
	}

	// Check for approved status (default priority)
	if decisionCounts["APPROVED"] > 0 {
		return "APPROVED"
	}

	// Check for rejected status
	if decisionCounts["REJECTED"] > 0 {
		return "REJECTED"
	}

	// Check for any other non-pending status
	for _, decision := range validDecisions {
		if decision != "PENDING" && decisionCounts[decision] > 0 {
			return decision
		}
	}

	return "PENDING"
}

// filterDocumentsByPendingApprovals filters documents that have pending approvals
func filterDocumentsByPendingApprovals(docs []*Document) []*Document {
	var pendingDocs []*Document
	for _, doc := range docs {
		for _, decision := range doc.ApprovalsMap {
			if decision.Status == Pending {
				pendingDocs = append(pendingDocs, doc)
				break
			}
		}
	}
	return pendingDocs
}

// calculateDocumentStatistics computes comprehensive statistics for documents
func (s *SmartContract) calculateDocumentStatistics(docs []*Document) map[string]interface{} {
	stats := map[string]interface{}{
		"totalDocuments":     len(docs),
		"statusCounts":       make(map[string]int),
		"uploaderCounts":     make(map[string]int),
		"workflowEnabled":    0,
		"deadlineEnabled":    0,
		"totalVersions":      0,
		"avgVersionsPerDoc":  0.0,
	}

	if len(docs) == 0 {
		return stats
	}

	statusCounts := make(map[string]int)
	uploaderCounts := make(map[string]int)
	totalVersions := 0
	workflowEnabled := 0
	deadlineEnabled := 0

	for _, doc := range docs {
		// Calculate and count by status
		docStatusCounts := calculateStatusCounts(doc.ApprovalsMap)
		docStatus := s.determineDocumentStatus(docStatusCounts, doc.ValidDecisions)
		statusCounts[docStatus]++

		// Count by uploader
		uploaderCounts[doc.Uploader]++

		// Count versions
		totalVersions += len(doc.Versions)

		// Count workflow enabled
		if doc.Workflow.Enabled {
			workflowEnabled++
		}

		// Count deadline enabled
		if doc.DeadlineConfig.Enabled {
			deadlineEnabled++
		}
	}

	stats["statusCounts"] = statusCounts
	stats["uploaderCounts"] = uploaderCounts
	stats["workflowEnabled"] = workflowEnabled
	stats["deadlineEnabled"] = deadlineEnabled
	stats["totalVersions"] = totalVersions
	stats["avgVersionsPerDoc"] = float64(totalVersions) / float64(len(docs))

	return stats
}

// createDocumentSummary creates a summary map for a document
func (s *SmartContract) createDocumentSummary(doc *Document) map[string]interface{} {
	statusCounts := calculateStatusCounts(doc.ApprovalsMap)
	currentStatus := s.determineDocumentStatus(statusCounts, doc.ValidDecisions)

	summary := map[string]interface{}{
		"id":                      doc.ID,
		"title":                   doc.Title,
		"description":             doc.Description,
		"uploader":                doc.Uploader,
		"currentStatus":           currentStatus,
		"latestVersion":           doc.LatestVersion,
		"createdTimestamp":        doc.CreatedTimestamp,
		"lastModifiedTimestamp":   doc.LastModifiedTimestamp,
		"totalApprovers":          len(doc.ApprovalsMap),
		"approvalCounts":          statusCounts,
		"editorsCount":            len(doc.Editors),
		"privilegedEditors":       len(doc.PrivilegedEditors),
		"workflowEnabled":         doc.Workflow.Enabled,
		"totalVersions":           len(doc.Versions),
		"validDecisions":          doc.ValidDecisions,
		"deadlineEnabled":         doc.DeadlineConfig.Enabled,
	}

	// Add workflow info if enabled
	if doc.Workflow.Enabled {
		summary["currentStage"] = doc.Workflow.CurrentStage
		summary["totalStages"] = len(doc.Workflow.Stages)
		summary["completedStages"] = len(doc.Workflow.CompletedStages)
	}

	// Add deadline info if enabled
	if doc.DeadlineConfig.Enabled {
		summary["documentDeadline"] = doc.DeadlineConfig.DocumentDeadline
	}

	return summary
}

// validateNonEmptyString validates that a string parameter is not empty
func validateNonEmptyString(paramName, value string) error {
	if value == "" {
		return fmt.Errorf("%s cannot be empty", paramName)
	}
	return nil
}