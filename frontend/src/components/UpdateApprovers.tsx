import React, { useState, useEffect } from 'react';
import { Form, Button, Container, Card, Alert, Spinner, ListGroup } from 'react-bootstrap';
import { useParams, useNavigate } from 'react-router-dom';
import { apiFetch } from '../utils/api';

const UpdateApprovers: React.FC = () => {
  const { documentId } = useParams<{ documentId: string }>();
  const [approvers, setApprovers] = useState<string[]>([]);
  const [currentApprover, setCurrentApprover] = useState('');
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

  const handleAddApprover = () => {
    if (currentApprover && !approvers.includes(currentApprover)) {
      setApprovers([...approvers, currentApprover]);
      setCurrentApprover('');
    }
  };

  const handleRemoveApprover = (approverToRemove: string) => {
    setApprovers(approvers.filter(approver => approver !== approverToRemove));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      await apiFetch(`/api/documents/${documentId}/approvers`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ newApprovers: approvers }),
      });
      setSuccess('Approvers updated successfully!');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container className="my-4">
      <Card className="p-4 shadow-sm">
        <Card.Body>
          <h2 className="text-center mb-4 text-primary">Update Approvers for {documentId}</h2>
          {error && <Alert variant="danger">{error}</Alert>}
          {success && <Alert variant="success">{success}</Alert>}
          <Form onSubmit={handleSubmit}>
            <Form.Group controlId="approvers" className="mb-3">
              <Form.Label>Add Approvers</Form.Label>
              <div className="d-flex">
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
                <Button variant="secondary" onClick={handleAddApprover} disabled={!currentApprover || loading || usersLoading} className="ms-2">
                  Add
                </Button>
              </div>
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

            <Button variant="primary" type="submit" disabled={loading || usersLoading}>
              {loading ? <Spinner animation="border" size="sm" /> : 'Update Approvers'}
            </Button>
            <Button variant="secondary" className="ms-2" onClick={() => navigate(`/view/${documentId}`)}>
              Back to Document
            </Button>
          </Form>
        </Card.Body>
      </Card>
    </Container>
  );
};

export default UpdateApprovers;