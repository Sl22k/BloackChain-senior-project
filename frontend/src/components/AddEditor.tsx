import React, { useState } from 'react';
import { Form, Button, Container, Card, Alert, Spinner } from 'react-bootstrap';
import { useParams, useNavigate } from 'react-router-dom';
import { apiFetch } from '../utils/api';

const AddEditor: React.FC = () => {
  const { documentId } = useParams<{ documentId: string }>();
  const [newEditor, setNewEditor] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setSuccess(null);

    try {
      await apiFetch(`/api/documents/${documentId}/editors`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ newEditor }),
      });
      setSuccess('Editor added successfully!');
      setNewEditor('');
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
          <h2 className="text-center mb-4 text-primary">Add Editor to {documentId}</h2>
          {error && <Alert variant="danger">{error}</Alert>}
          {success && <Alert variant="success">{success}</Alert>}
          <Form onSubmit={handleSubmit}>
            <Form.Group controlId="newEditor" className="mb-3">
              <Form.Label>New Editor's Email</Form.Label>
              <Form.Control
                type="email"
                placeholder="Enter editor's email"
                value={newEditor}
                onChange={(e) => setNewEditor(e.target.value)}
                required
                disabled={loading}
              />
            </Form.Group>
            <Button variant="primary" type="submit" disabled={loading}>
              {loading ? <Spinner animation="border" size="sm" /> : 'Add Editor'}
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

export default AddEditor;