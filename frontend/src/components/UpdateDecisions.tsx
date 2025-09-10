import React, { useState, useEffect } from 'react';
import { Form, Button, Container, Card, Alert, Spinner, ListGroup, Row, Col } from 'react-bootstrap';
import { useParams, useNavigate } from 'react-router-dom';
import { apiFetch } from '../utils/api';

const UpdateDecisions: React.FC = () => {
  const { documentId } = useParams<{ documentId: string }>();
  const navigate = useNavigate();

  const [validDecisions, setValidDecisions] = useState<string[]>([]);
  const [currentDecision, setCurrentDecision] = useState('');
  const availableDecisions = ['APPROVED', 'REJECTED', 'NEEDS_REVISION', 'MORE_INFO_REQUIRED'];
  
  const [initialLoading, setInitialLoading] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    const fetchCurrentDecisions = async () => {
      if (!documentId) return;
      try {
        const data = await apiFetch(`/api/documents/${documentId}`);
        if (data.onChain?.ValidDecisions) {
          setValidDecisions(data.onChain.ValidDecisions);
        }
      } catch (err: any) {
        setError(err.message);
      } finally {
        setInitialLoading(false);
      }
    };
    fetchCurrentDecisions();
  }, [documentId]);

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

    try {
      // Assuming this is the correct endpoint and method
      await apiFetch(`/api/documents/${documentId}/decisions`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ validDecisions }),
      });
      setSuccess('Valid decisions updated successfully!');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  if (initialLoading) {
    return <Spinner animation="border" className="d-block mx-auto my-5" />;
  }

  return (
    <Container className="my-4">
      <Card className="p-4 shadow-sm">
        <Card.Body>
          <h2 className="text-center mb-4 text-primary">Update Valid Decisions for {documentId}</h2>
          {error && <Alert variant="danger">{error}</Alert>}
          {success && <Alert variant="success">{success}</Alert>}
          <Form onSubmit={handleSubmit}>
            <Form.Group controlId="validDecisions" className="mb-3">
              <Form.Label>Manage Valid Decisions</Form.Label>
              <Row>
                <Col xs={9}>
                  <Form.Select
                    value={currentDecision}
                    onChange={(e) => setCurrentDecision(e.target.value)}
                    disabled={loading}
                  >
                    <option value="">Select a decision to add...</option>
                    {availableDecisions.map((decision) => (
                      <option key={decision} value={decision}>
                        {decision}
                      </option>
                    ))}
                  </Form.Select>
                </Col>
                <Col xs={3}>
                  <Button variant="secondary" onClick={handleAddDecision} disabled={!currentDecision || loading}>
                    Add Decision
                  </Button>
                </Col>
              </Row>
            </Form.Group>

            {validDecisions.length > 0 && (
              <div className="mb-3">
                <h5>Current Valid Decisions:</h5>
                <ListGroup>
                  {validDecisions.map((decision) => (
                    <ListGroup.Item key={decision} className="d-flex justify-content-between align-items-center">
                      {decision}
                      <Button variant="danger" size="sm" onClick={() => handleRemoveDecision(decision)} disabled={loading}>
                        Remove
                      </Button>
                    </ListGroup.Item>
                  ))}
                </ListGroup>
              </div>
            )}
            
            <Button variant="primary" type="submit" disabled={loading}>
              {loading ? <Spinner animation="border" size="sm" /> : 'Update Decisions'}
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

export default UpdateDecisions;
