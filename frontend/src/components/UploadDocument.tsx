import React, { useState, useEffect } from 'react';
import { Form, Button, Container, Row, Col, Alert, Spinner, ListGroup, Card } from 'react-bootstrap';
import { useNavigate } from 'react-router-dom';
import { sha256 } from 'js-sha256';
import { apiFetch } from '../utils/api';

interface UploadDocumentProps {
  loggedInUser: string | null;
  userEmail: string | null;
}

const UploadDocument: React.FC<UploadDocumentProps> = ({ loggedInUser, userEmail }) => {
  const [file, setFile] = useState<File | null>(null);
  const [documentId, setDocumentId] = useState('');
  const [approvers, setApprovers] = useState<string[]>([]);
  const [currentApprover, setCurrentApprover] = useState('');
  const [validDecisions, setValidDecisions] = useState<string[]>([]);
  const [currentDecision, setCurrentDecision] = useState('');
  const availableDecisions = ['APPROVED', 'REJECTED', 'NEEDS_REVISION', 'MORE_INFO_REQUIRED'];
  const [resetApprovals, setResetApprovals] = useState(false);
  const [allUsers, setAllUsers] = useState<any[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const fetchUsers = async () => {
      try {
        const data = await apiFetch('/api/users');
        setAllUsers(data);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setUsersLoading(false);
      }
    };
    fetchUsers();
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      const fileName = e.target.files[0].name;
      const id = fileName.substring(0, fileName.lastIndexOf('.')) || fileName;
      setDocumentId(id);
    }
  };

  const handleAddApprover = () => {
    if (currentApprover && !approvers.includes(currentApprover)) {
      setApprovers([...approvers, currentApprover]);
      setCurrentApprover('');
    }
  };

  const handleRemoveApprover = (approverToRemove: string) => {
    setApprovers(approvers.filter(approver => approver !== approverToRemove));
  };

  const handleAddDecision = () => {
    if (currentDecision && !validDecisions.includes(currentDecision)) {
      setValidDecisions([...validDecisions, currentDecision]);
      setCurrentDecision('');
    }
  };

  const handleRemoveDecision = (decisionToRemove: string) => {
    setValidDecisions(validDecisions.filter(decision => decision !== decisionToRemove));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(null);

    if (!file || !documentId || approvers.length === 0) {
      setError('Please select a file, provide a document ID, and add at least one approver.');
      setLoading(false);
      return;
    }

    try {
      const fileContent = await file.text();
      const fileHash = sha256(fileContent);

      if (!loggedInUser) {
        setError('Could not find your user details to verify uploader. Please try refreshing the page.');
        setLoading(false);
        return;
      }

      const uploader = loggedInUser;

      const formData = new FormData();
      formData.append('file', file);

      await apiFetch('/api/documents/upload', {
        method: 'POST',
        body: formData,
      });

      const docData = {
        id: documentId,
        name: file.name,
        hash: fileHash,
        uploader: uploader,
        approvers: approvers,
        validDecisions: validDecisions,
        resetApprovals: resetApprovals,
      };

      await apiFetch('/api/documents', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(docData),
      });

      setSuccess('Document uploaded and submitted successfully!');
      setFile(null);
      setDocumentId('');
      setApprovers([]);
      setCurrentApprover('');
      setValidDecisions([]);
      setResetApprovals(false);
    } catch (err: any) {
      setError(err.message || 'An unknown error occurred during upload.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container className="my-4">
      <Card className="p-4 shadow-sm">
        <Card.Body>
          <h2 className="text-center mb-4 text-primary">Upload New Document</h2>
          {error && <Alert variant="danger">{error}</Alert>}
          {success ? (
            <Alert variant="success">
              {success}
              <hr />
              <div className="d-flex justify-content-end">
                <Button onClick={() => navigate('/uploaded', { state: { from: 'upload' } })} variant="outline-success">
                  View Uploaded Documents
                </Button>
              </div>
            </Alert>
          ) : (
            <Form onSubmit={handleSubmit}>
              <Form.Group controlId="formFile" className="mb-3">
                <Form.Label>Select Document File</Form.Label>
                <Form.Control type="file" onChange={handleFileChange} required disabled={loading || usersLoading} />
              </Form.Group>

              <Form.Group controlId="documentId" className="mb-3">
                <Form.Label>Document ID</Form.Label>
                <Form.Control
                  type="text"
                  placeholder="Enter unique document ID"
                  value={documentId}
                  onChange={(e) => setDocumentId(e.target.value)}
                  required
                  disabled={loading || usersLoading}
                />
              </Form.Group>

              <Form.Group controlId="approvers" className="mb-3">
                <Form.Label>Add Approvers</Form.Label>
                <Row>
                  <Col xs={9}>
                    <Form.Select
                      value={currentApprover}
                      onChange={(e) => setCurrentApprover(e.target.value)}
                      disabled={loading || usersLoading}
                    >
                      <option value="">Select an approver...</option>
                      {allUsers.map((user) => (
                        <option key={user.id} value={user.email}>
                          {user.username} ({user.email})
                        </option>
                      ))}
                    </Form.Select>
                  </Col>
                  <Col xs={3}>
                    <Button variant="secondary" onClick={handleAddApprover} disabled={!currentApprover || loading || usersLoading}>
                      Add Approver
                    </Button>
                  </Col>
                </Row>
              </Form.Group>

              {approvers.length > 0 && (
                <div className="mb-3">
                  <h5>Selected Approvers:</h5>
                  <ListGroup>
                    {approvers.map((approver) => (
                      <ListGroup.Item key={approver} className="d-flex justify-content-between align-items-center">
                        {allUsers.find(u => u.email === approver)?.username || approver}
                        <Button variant="danger" size="sm" onClick={() => handleRemoveApprover(approver)} disabled={loading || usersLoading}>
                          Remove
                        </Button>
                      </ListGroup.Item>
                    ))}
                  </ListGroup>
                </div>
              )}

              <Form.Group controlId="validDecisions" className="mb-3">
                <Form.Label>Add Valid Decisions</Form.Label>
                <Row>
                  <Col xs={9}>
                    <Form.Select
                      value={currentDecision}
                      onChange={(e) => setCurrentDecision(e.target.value)}
                      disabled={loading || usersLoading}
                    >
                      <option value="">Select a decision...</option>
                      {availableDecisions.map((decision) => (
                        <option key={decision} value={decision}>
                          {decision}
                        </option>
                      ))}
                    </Form.Select>
                  </Col>
                  <Col xs={3}>
                    <Button variant="secondary" onClick={handleAddDecision} disabled={!currentDecision || loading || usersLoading}>
                      Add Decision
                    </Button>
                  </Col>
                </Row>
              </Form.Group>

              {validDecisions.length > 0 && (
                <div className="mb-3">
                  <h5>Selected Decisions:</h5>
                  <ListGroup>
                    {validDecisions.map((decision) => (
                      <ListGroup.Item key={decision} className="d-flex justify-content-between align-items-center">
                        {decision}
                        <Button variant="danger" size="sm" onClick={() => handleRemoveDecision(decision)} disabled={loading || usersLoading}>
                          Remove
                        </Button>
                      </ListGroup.Item>
                    ))}
                  </ListGroup>
                </div>
              )}

              <Form.Group controlId="resetApprovals" className="mb-3">
                <Form.Check
                  type="checkbox"
                  label="Reset approvals if submitting a new version of an existing document"
                  checked={resetApprovals}
                  onChange={(e) => setResetApprovals(e.target.checked)}
                  disabled={loading || usersLoading}
                />
              </Form.Group>

              <Button variant="primary" type="submit" disabled={loading || usersLoading}>
                {loading || usersLoading ? <Spinner animation="border" size="sm" /> : 'Upload and Submit'}
              </Button>
            </Form>
          )}
        </Card.Body>
      </Card>
    </Container>
  );
};

export default UploadDocument;