import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Spinner, Alert, Button, Card, ButtonGroup, Row, Col, Form, Modal, ListGroup, Badge, Accordion } from 'react-bootstrap';
import { LinkContainer } from 'react-router-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCheck, faTimes, faPlus, faUserEdit, faTasks } from '@fortawesome/free-solid-svg-icons';
import { apiFetch, apiFetchFile } from '../utils/api';

// Define interfaces
interface Decision {
  Status: string;
  Comment: string;
}
interface DocumentVersion {
  Version: number;
  Hash: string;
  Submitter: string;
  Timestamp: number;
  ApprovalsMap?: { [key: string]: Decision };
  ValidDecisions?: string[];
  ApproversChanged?: boolean;
  DecisionsChanged?: boolean;
  WorkflowSnapshot?: any;
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
  loggedInUser: string | null;
}

const DocumentViewer: React.FC<DocumentViewerProps> = ({ userRole, loggedInUser }) => {
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
  
  const [allUsers, setAllUsers] = useState<any[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);

  const [currentEditor, setCurrentEditor] = useState('');
  const [addEditorLoading, setAddEditorLoading] = useState(false);
  const [addEditorSuccess, setAddEditorSuccess] = useState<string | null>(null);
  const [addEditorError, setAddEditorError] = useState<string | null>(null);

  const [currentPrivilegedEditor, setCurrentPrivilegedEditor] = useState('');
  const [addPrivilegedEditorLoading, setAddPrivilegedEditorLoading] = useState(false);
  const [addPrivilegedEditorSuccess, setAddPrivilegedEditorSuccess] = useState<string | null>(null);
  const [addPrivilegedEditorError, setAddPrivilegedEditorError] = useState<string | null>(null);

  const [showUploadModal, setShowUploadModal] = useState(false);
  const [fileToUpload, setFileToUpload] = useState<File | null>(null);

  useEffect(() => {
    const fetchInitialData = async () => {
      if (!documentId) return;
      setLoading(true);
      setUsersLoading(true);
      try {
        const data: FullDocument = await apiFetch(`/api/documents/${documentId}`);
        setDocument(data);

        if (data.offChain?.doc_name) {
          const blob = await apiFetchFile(`http://localhost:8080/api/documents/content/${data.offChain.doc_name}`);
          const url = URL.createObjectURL(blob);
          setDocumentUrl(url);
        }

        const usersData = await apiFetch('/api/users');
        setAllUsers(usersData);

      } catch (err: any) {
        setError(`Network error: ${err.message}`);
      } finally {
        setLoading(false);
        setUsersLoading(false);
      }
    };

    fetchInitialData();
  }, [documentId]);

  const refreshDocument = async () => {
    if (!documentId) return;
    const data: FullDocument = await apiFetch(`/api/documents/${documentId}`);
    setDocument(data);
  };

  const handleAddEditor = async () => {
    if (!currentEditor || !documentId) return;
    setAddEditorLoading(true);
    setAddEditorError(null);
    setAddEditorSuccess(null);
    try {
      await apiFetch(`/api/documents/${documentId}/editors`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ newEditor: currentEditor }),
      });
      setAddEditorSuccess(`Editor ${currentEditor} added successfully! They have been notified.`);
      setCurrentEditor('');
      await refreshDocument();
    } catch (err: any) {
      setAddEditorError(err.message || 'Failed to add editor');
    } finally {
      setAddEditorLoading(false);
    }
  };

  const handleAddPrivilegedEditor = async () => {
    if (!currentPrivilegedEditor || !documentId) return;
    setAddPrivilegedEditorLoading(true);
    setAddPrivilegedEditorError(null);
    setAddPrivilegedEditorSuccess(null);
    try {
      await apiFetch(`/api/documents/${documentId}/privileged`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ newPrivilegedEditor: currentPrivilegedEditor }),
      });
      setAddPrivilegedEditorSuccess(`Privileged editor ${currentPrivilegedEditor} added successfully! They have been notified.`);
      setCurrentPrivilegedEditor('');
      await refreshDocument();
    } catch (err: any) {
      setAddPrivilegedEditorError(err.message || 'Failed to add privileged editor');
    } finally {
      setAddPrivilegedEditorLoading(false);
    }
  };

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

  const handleUpload = async (keepApprovals: boolean) => {
    if (!fileToUpload || !documentId) return;
    setActionLoading(true);
    setError(null);
    try {
      const formData = new FormData();
      formData.append('file', fileToUpload);
      formData.append('keepApprovals', String(keepApprovals));
      formData.append('uploader', loggedInUser || '');

      // I'm assuming this is the endpoint for uploading a new version.
      await apiFetch(`/api/documents/${documentId}/versions`, {
        method: 'POST',
        body: formData,
      });

      setShowUploadModal(false);
      setFileToUpload(null);
      // Refresh the document to show the new version
      await refreshDocument();
    } catch (err: any) {
      setError(`Failed to upload new version: ${err.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  if (loading) return <Spinner animation="border" className="d-block mx-auto my-5" />;
  if (error) return <Alert variant="danger" className="my-3">{error}</Alert>;
  if (!document) return <Alert variant="warning">Document not found.</Alert>;

  const { onChain, offChain } = document;
  const isUploader = onChain.Uploader && loggedInUser === onChain.Uploader;
  const isPrivilegedEditor = onChain.PrivilegedEditors && loggedInUser ? onChain.PrivilegedEditors.includes(loggedInUser) : false;
  const isEditor = onChain.Editors && loggedInUser ? onChain.Editors.includes(loggedInUser) : false;
  const canApprove = onChain.ApprovalsMap && loggedInUser && onChain.ApprovalsMap[loggedInUser]?.Status === 'PENDING' && userRole !== 'student';
  const canUploadNewVersion = isUploader || isPrivilegedEditor || isEditor;
  const isTrainingLetter = documentId?.startsWith('training-letter-');

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
          {!isTrainingLetter &&
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
          }

          <Card className="mb-4">
            <Card.Header>Editors</Card.Header>
            <ListGroup variant="flush">
              {onChain.Editors?.map(editor => <ListGroup.Item key={editor}>{editor}</ListGroup.Item>)}
            </ListGroup>
            {isUploader && (
              <Card.Footer>
                <Form.Group>
                  <Row>
                    <Col xs={9}>
                      <Form.Select value={currentEditor} onChange={(e) => setCurrentEditor(e.target.value)} disabled={usersLoading || addEditorLoading}>
                        <option value="">Select an editor...</option>
                        {allUsers.filter(u => !onChain.Editors?.includes(u.username) && !onChain.PrivilegedEditors?.includes(u.username)).map((user: any) => (
                          <option key={user.id} value={user.username}>{user.username}</option>
                        ))}
                      </Form.Select>
                    </Col>
                    <Col xs={3}>
                      <Button onClick={handleAddEditor} disabled={!currentEditor || addEditorLoading}>{addEditorLoading ? <Spinner size="sm" /> : 'Add'}</Button>
                    </Col>
                  </Row>
                </Form.Group>
                {addEditorSuccess && <Alert variant="success" className="mt-2">{addEditorSuccess}</Alert>}
                {addEditorError && <Alert variant="danger" className="mt-2">{addEditorError}</Alert>}
              </Card.Footer>
            )}
          </Card>
        </Col>

        <Col md={6}>
          <Card className="mb-4">
            <Card.Header>Version History (Latest Version: {onChain.LatestVersion})</Card.Header>
            <Accordion>
              {onChain.Versions?.slice().reverse().map((v, index, reversedVersions) => {
                let versionApprovals: { [key: string]: Decision } | null | undefined = null;

                if (v.Version === onChain.LatestVersion) {
                  // For the latest version, show the current top-level approval status.
                  versionApprovals = onChain.ApprovalsMap;
                } else {
                  // For older versions, use the workaround for the backend bug.
                  const approvalSourceVersion = reversedVersions[index - 1];
                  versionApprovals = approvalSourceVersion ? approvalSourceVersion.ApprovalsMap : null;
                }

                return (
                  <Accordion.Item eventKey={String(index)} key={v.Version}>
                    <Accordion.Header>Version {v.Version} - by {v.Submitter}</Accordion.Header>
                    <Accordion.Body>
                      <p><strong>Hash:</strong> {v.Hash}</p>
                      <p><strong>Date:</strong> {new Date(v.Timestamp * 1000).toLocaleString()}</p>
                      {versionApprovals && Object.keys(versionApprovals).length > 0 && (
                        <div>
                          <h6>Approvals for this version:</h6>
                          <ListGroup>
                            {Object.entries(versionApprovals).map(([approver, decision]) => (
                              <ListGroup.Item key={approver}>
                                <strong>{approver}:</strong> {decision.Status}
                                {decision.Comment && <p className="mb-0 mt-1 text-muted fst-italic">Comment: "{decision.Comment}"</p>}
                              </ListGroup.Item>
                            ))}
                          </ListGroup>
                        </div>
                      )}
                    </Accordion.Body>
                  </Accordion.Item>
                );
              })}
            </Accordion>
            {canUploadNewVersion && (
              <Card.Footer>
                <Button variant="primary" onClick={() => setShowUploadModal(true)}>Upload New Version</Button>
              </Card.Footer>
            )}
          </Card>

          <Card className="mb-4">
            <Card.Header>Privileged Editors</Card.Header>
            <ListGroup variant="flush">
              {onChain.PrivilegedEditors?.map(editor => <ListGroup.Item key={editor}>{editor}</ListGroup.Item>)}
            </ListGroup>
            <Card.Footer>
              {isUploader && (
                <Form.Group className="mb-2">
                  <Row>
                    <Col xs={9}>
                      <Form.Select value={currentPrivilegedEditor} onChange={(e) => setCurrentPrivilegedEditor(e.target.value)} disabled={usersLoading || addPrivilegedEditorLoading}>
                        <option value="">Select a privileged editor...</option>
                        {allUsers.filter(u => !onChain.Editors?.includes(u.username) && !onChain.PrivilegedEditors?.includes(u.username)).map((user: any) => (
                          <option key={user.id} value={user.username}>{user.username}</option>
                        ))}
                      </Form.Select>
                    </Col>
                    <Col xs={3}>
                      <Button onClick={handleAddPrivilegedEditor} disabled={!currentPrivilegedEditor || addPrivilegedEditorLoading}>{addPrivilegedEditorLoading ? <Spinner size="sm" /> : 'Add'}</Button>
                    </Col>
                  </Row>
                </Form.Group>
              )}
              {addPrivilegedEditorSuccess && <Alert variant="success" className="mt-2">{addPrivilegedEditorSuccess}</Alert>}
              {addPrivilegedEditorError && <Alert variant="danger" className="mt-2">{addPrivilegedEditorError}</Alert>}
              <hr />
              <div className="d-flex justify-content-between">
                <LinkContainer to={`/documents/${documentId}/update-approvers`}>
                  <Button variant="outline-secondary" size="sm" disabled={!isUploader}><FontAwesomeIcon icon={faUserEdit} className="me-2" />Update Approvers</Button>
                </LinkContainer>
                {isPrivilegedEditor && (
                  <LinkContainer to={`/documents/${documentId}/update-decisions`}>
                    <Button variant="outline-info" size="sm"><FontAwesomeIcon icon={faTasks} className="me-2" />Update Decisions</Button>
                  </LinkContainer>
                )}
              </div>
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

      <Modal show={showUploadModal} onHide={() => setShowUploadModal(false)}>
        <Modal.Header closeButton>
          <Modal.Title>Upload New Version</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <p>Would you like to keep the approval of the previous version to the new version or reset?</p>
          <Form.Group controlId="formFile" className="mb-3">
            <Form.Label>Select a file to upload</Form.Label>
            <Form.Control type="file" onChange={(e: React.ChangeEvent<HTMLInputElement>) => setFileToUpload(e.target.files ? e.target.files[0] : null)} />
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="secondary" onClick={() => handleUpload(false)} disabled={!fileToUpload || actionLoading}>
            Upload and Reset Approvals
          </Button>
          <Button variant="primary" onClick={() => handleUpload(true)} disabled={!fileToUpload || actionLoading}>
            Upload and Keep Approvals
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default DocumentViewer;
