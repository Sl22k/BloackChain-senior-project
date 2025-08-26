import React, { useState, useEffect, useRef } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { Spinner, Alert, Button, Card, ButtonGroup, Row, Col, Form, Modal, ListGroup, Badge } from 'react-bootstrap';
import { LinkContainer } from 'react-router-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCheck, faTimes, faFilePdf, faPlus, faUserEdit } from '@fortawesome/free-solid-svg-icons';
import { apiFetch, apiFetchFile } from '../utils/api';

// Define interfaces for our complex document object
interface Decision {
  Status: string;
  Comment: string;
}

interface DocumentVersion {
  Version: number;
  Hash: string;
  Submitter: string;
  Timestamp: number;
}

interface OnChainData {
  ID: string;
  LatestVersion: number;
  Versions: DocumentVersion[];
  Uploader: string;
  PrivilegedEditors: string[];
  Editors: string[];
  ApprovalsMap: { [key: string]: Decision };
  ValidDecisions: string[];
}

interface OffChainData {
  doc_name: string;
  doc_path: string;
}

interface FullDocument {
  onChain: OnChainData;
  offChain: OffChainData;
}

interface DocumentViewerProps {
  userRole: string | null;
}

const DocumentViewer: React.FC<DocumentViewerProps> = ({ userRole }) => {
  const { documentId } = useParams<{ documentId: string }>();
  const navigate = useNavigate();
  const [document, setDocument] = useState<FullDocument | null>(null);
  const [documentUrl, setDocumentUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [showApprovalModal, setShowApprovalModal] = useState(false);
  const [approvalDecision, setApprovalDecision] = useState<'APPROVED' | 'REJECTED'>('APPROVED');
  const [approvalComment, setApprovalComment] = useState('');

  useEffect(() => {
    const fetchDocumentData = async () => {
      if (!documentId) return;
      setLoading(true);
      try {
        const data: FullDocument = await apiFetch(`/api/documents/${documentId}`);
        setDocument(data);

        if (data.offChain?.doc_name) {
          const blob = await apiFetchFile(`http://localhost:8080/api/documents/content/${data.offChain.doc_name}`);
          const url = URL.createObjectURL(blob);
          setDocumentUrl(url);
        }
      } catch (err: any) {
        setError(`Network error: ${err.message}`);
      } finally {
        setLoading(false);
      }
    };

    fetchDocumentData();
  }, [documentId]);

  const handleShowApprovalModal = (decision: 'APPROVED' | 'REJECTED') => {
    setApprovalDecision(decision);
    setShowApprovalModal(true);
  };

  const handleApprovalSubmit = async () => {
    if (!documentId) return;
    setActionLoading(true);
    try {
      await apiFetch(`/api/documents/${documentId}/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ decision: approvalDecision, comment: approvalComment }),
      });
      setShowApprovalModal(false);
      setApprovalComment('');
      navigate('/uploaded', { state: { from: 'approval', timestamp: Date.now() } });
    } catch (err: any) {
      setError(`Failed to submit approval: ${err.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) {
    return <Spinner animation="border" className="d-block mx-auto my-5" />;
  }

  if (error) {
    return <Alert variant="danger" className="my-3">{error}</Alert>;
  }

  if (!document) {
    return <Alert variant="warning">Document not found.</Alert>;
  }

  const { onChain, offChain } = document;
  const loggedInUser = JSON.parse(localStorage.getItem('user') || '{}').username;
  const canApprove = onChain.ApprovalsMap && loggedInUser && onChain.ApprovalsMap[loggedInUser]?.Status === 'PENDING';

  return (
    <>
      <Card className="my-4 shadow-sm">
        <Card.Header as="h2" className="text-center bg-primary text-white">
          {offChain?.doc_name || onChain.ID}
        </Card.Header>
        <Card.Body>
          {documentUrl ? (
            <iframe src={documentUrl} title="Document Viewer" width="100%" height="600px" style={{ border: '1px solid #ccc' }} />
          ) : (
            <Alert variant="info">No document preview available.</Alert>
          )}

          {canApprove && (
            <div className="mt-4 text-center">
              <h4>Take Action</h4>
              <ButtonGroup aria-label="Document Actions">
                <Button variant="success" onClick={() => handleShowApprovalModal('APPROVED')} disabled={actionLoading}>
                  <FontAwesomeIcon icon={faCheck} className="me-2" />Approve
                </Button>
                <Button variant="danger" onClick={() => handleShowApprovalModal('REJECTED')} disabled={actionLoading}>
                  <FontAwesomeIcon icon={faTimes} className="me-2" />Reject
                </Button>
              </ButtonGroup>
            </div>
          )}
        </Card.Body>
      </Card>

      <Row>
        <Col md={6}>
          <Card className="mb-4">
            <Card.Header>Approval Status</Card.Header>
            <ListGroup variant="flush">
              {Object.entries(onChain.ApprovalsMap)?.map(([approver, decision]) => (
                <ListGroup.Item key={approver}>
                  <div className="d-flex justify-content-between">
                    <strong>{approver}</strong>
                    <Badge bg={decision.Status === 'APPROVED' ? 'success' : decision.Status === 'REJECTED' ? 'danger' : 'secondary'}>
                      {decision.Status}
                    </Badge>
                  </div>
                  {decision.Comment && <p className="mb-0 mt-1 text-muted fst-italic">Comment: "{decision.Comment}"</p>}
                </ListGroup.Item>
              ))}
            </ListGroup>
          </Card>

          <Card className="mb-4">
            <Card.Header>Editors</Card.Header>
            <ListGroup variant="flush">
              {onChain.Editors?.map(editor => <ListGroup.Item key={editor}>{editor}</ListGroup.Item>)}
            </ListGroup>
            <Card.Footer>
              {(userRole === 'faculty' || userRole === 'coordinator') && (
                <LinkContainer to={`/documents/${documentId}/add-editor`}>
                  <Button variant="outline-secondary" size="sm">
                    <FontAwesomeIcon icon={faPlus} className="me-2" />Add Editor
                  </Button>
                </LinkContainer>
              )}
            </Card.Footer>
          </Card>
        </Col>

        <Col md={6}>
          <Card className="mb-4">
            <Card.Header>Version History (Latest Version: {onChain.LatestVersion})</Card.Header>
            <ListGroup variant="flush">
              {onChain.Versions?.slice().reverse().map(v => (
                <ListGroup.Item key={v.Version}>
                  <strong>Version {v.Version}</strong> by {v.Submitter}
                  <div className="text-muted small">Hash: {v.Hash}</div>
                  <div className="text-muted small">Date: {new Date(v.Timestamp * 1000).toLocaleString()}</div>
                </ListGroup.Item>
              ))}
            </ListGroup>
          </Card>

          <Card className="mb-4">
            <Card.Header>Privileged Editors</Card.Header>
            <ListGroup variant="flush">
              {onChain.PrivilegedEditors?.map(editor => <ListGroup.Item key={editor}>{editor}</ListGroup.Item>)}
            </ListGroup>
            <Card.Footer>
              {(userRole === 'faculty' || userRole === 'coordinator') && (
                <LinkContainer to={`/documents/${documentId}/update-approvers`}>
                  <Button variant="outline-secondary" size="sm">
                    <FontAwesomeIcon icon={faUserEdit} className="me-2" />Update Approvers
                  </Button>
                </LinkContainer>
              )}
            </Card.Footer>
          </Card>
        </Col>
      </Row>

      <Modal show={showApprovalModal} onHide={() => setShowApprovalModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Confirm {approvalDecision}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form.Group>
            <Form.Label>Comment</Form.Label>
            <Form.Control as="textarea" rows={3} value={approvalComment} onChange={(e) => setApprovalComment(e.target.value)} />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => setShowApprovalModal(false)}>Cancel</Button>
          <Button variant="primary" onClick={handleApprovalSubmit} disabled={actionLoading}>
            {actionLoading ? <Spinner as="span" animation="border" size="sm" /> : 'Submit'}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default DocumentViewer;
