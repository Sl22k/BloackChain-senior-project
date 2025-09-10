import { Link } from 'react-router-dom';
import React, { useState, useEffect, useCallback } from 'react';
import { Table, Button, Spinner, Alert } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faEye } from '@fortawesome/free-solid-svg-icons';
import OverlayTrigger from 'react-bootstrap/OverlayTrigger';
import Tooltip from 'react-bootstrap/Tooltip';
import { apiFetch } from '../utils/api';

interface Document {
  doc_id: string;
  doc_name: string;
  sender_username: string;
  status: string;
  upload_time?: string;
  doc_path?: string;
  hash?: string;
}

const DocumentsToEdit: React.FC = () => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchDocuments = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const user = JSON.parse(localStorage.getItem('user') || '{}');
      const username = user.username || user.id;
      // This is an assumed endpoint. The backend needs to implement this.
      const data = await apiFetch(`/api/documents/editable?username=${encodeURIComponent(username)}`);
      setDocuments(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error("[FRONTEND] Fetch error:", err);
      setError(err instanceof Error ? err.message : String(err));
      setDocuments([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDocuments();
  }, [fetchDocuments]);

  if (loading) {
    return (
      <div className="text-center my-5">
        <Spinner animation="border" role="status">
          <span className="visually-hidden">Loading documents...</span>
        </Spinner>
        <p className="mt-2">Loading documents you can edit...</p>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="danger" className="my-3">
        <Alert.Heading>Error loading documents</Alert.Heading>
        <p>{error}</p>
        <Button variant="primary" onClick={fetchDocuments}>
          Try Again
        </Button>
      </Alert>
    );
  }

  return (
    <div className="container mt-4">
      <h2 className="mb-4 text-primary text-center">Documents to Edit</h2>
      
      {documents.length === 0 ? (
        <Alert variant="info" className="my-3 text-center">
          There are no documents assigned to you for editing.
        </Alert>
      ) : (
        <div className="table-responsive">
          <Table striped bordered hover className="shadow-sm">
            <thead className="bg-primary text-white">
              <tr>
                <th>Document Name</th>
                <th>Sender</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {documents.map((doc) => (
                <tr key={doc.doc_id}>
                  <td>
                    <div>{doc.doc_name}</div>
                    {doc.hash && (
                      <OverlayTrigger
                        placement="top"
                        overlay={<Tooltip id={`hash-tooltip-${doc.doc_id}`}>View Document Hash</Tooltip>}
                      >
                        <small className="text-muted">{`Hash: ${doc.hash}`}</small>
                      </OverlayTrigger>
                    )}
                  </td>
                  <td>{doc.sender_username}</td>
                  <td>
                    <span className={`badge rounded-pill ${
                      doc.status === 'APPROVED' ? 'bg-success' :
                      doc.status === 'REJECTED' ? 'bg-danger' :
                      'bg-warning text-dark'
                    }`}>
                      {doc.status}
                    </span>
                  </td>
                  <td>
                    <div className="d-flex justify-content-center">
                      <Link to={`/view/${doc.doc_id}`}>
                        <OverlayTrigger placement="top" overlay={<Tooltip id={`view-tooltip-${doc.doc_id}`}>View Document</Tooltip>}>
                          <Button variant="outline-primary" size="sm" className="me-2">
                            <FontAwesomeIcon icon={faEye} />
                          </Button>
                        </OverlayTrigger>
                      </Link>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
          <div className="text-muted small mt-2 text-center">
            Showing {documents.length} document{documents.length !== 1 ? 's' : ''}
          </div>
        </div>
      )}
    </div>
  );
};

export default DocumentsToEdit;
