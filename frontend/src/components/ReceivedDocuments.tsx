import { Link } from 'react-router-dom';
import React, { useState, useEffect, useCallback } from 'react';
import { Table, Button, Spinner, Alert } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCheck , faTimes , faEye} from '@fortawesome/free-solid-svg-icons';
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
  ApprovedCount : number;
  RejectedCount : number;
  pendingCount : number;
}

interface ReceivedDocumentsProps {
  userRole: string | null;
}

const ReceivedDocuments: React.FC<ReceivedDocumentsProps> = ({ userRole }) => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const isLatest = true; // or true, depending on your logic
  const fetchDocuments = useCallback(async (forceRefresh = false) => {
    console.log("[FRONTEND] Starting fetch...");
    setLoading(true);
    setError(null);
    if (forceRefresh) {
      setRefreshing(true);
    }

    try {
      const user = JSON.parse(localStorage.getItem('user') || '{}');
      const username = user.username || user.id;
      console.log("[FRONTEND] Using username:", username);

      const apiUrl = isLatest? `/api/documents/received/latest?username=${encodeURIComponent(username)}` : `/api/documents/received?username=${encodeURIComponent(username)}`;
      console.log("[FRONTEND] Request URL:", apiUrl);

      const startTime = performance.now();
      const data = await apiFetch(apiUrl);
      const duration = performance.now() - startTime;

      console.log(`[FRONTEND] Response received in ${duration.toFixed(2)}ms`);

      if (data.debug) {
        console.warn("[FRONTEND] Debug info from backend:", data.debug);
      }

      setDocuments(Array.isArray(data) ? data : []);
    } catch (err) {
      console.error("[FRONTEND] Fetch error:", err);
      setError(err instanceof Error ? err.message : String(err));
      setDocuments([]);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [isLatest]);

  useEffect(() => {
    fetchDocuments();
  }, [fetchDocuments]);

  const handleApproval = async (documentId: string, decision: string) => {
    try {
      setError(null);
      setRefreshing(true); // Set refreshing to true when approval starts
      const user = JSON.parse(localStorage.getItem('user') || '{}');
      const username = user.username || user.id;

      const apiUrl = `/api/documents/${documentId}/approve`;
      console.log("Approval API URL:", apiUrl);
      const response = await fetch(apiUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
        },
        body: JSON.stringify({ 
          approver: username,
          decision 
        }),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText);
      }

      // Refresh documents with loading state
      await fetchDocuments(true);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setRefreshing(false); // Set refreshing to false when approval finishes
    }
  };

  const handleRefresh = () => {
    fetchDocuments(true);
  };

  if (loading) {
    return (
      <div className="text-center my-5">
        <Spinner animation="border" role="status">
          <span className="visually-hidden">Loading documents...</span>
        </Spinner>
        <p className="mt-2">Loading your received documents...</p>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="danger" className="my-3">
        <Alert.Heading>Error loading documents</Alert.Heading>
        <p>{error}</p>
        <Button variant="primary" onClick={handleRefresh} disabled={refreshing}>
          {refreshing ? 'Refreshing...' : 'Try Again'}
        </Button>
      </Alert>
    );
  }

  return (
    <div className="container mt-4">
      <h2 className="mb-4 text-primary text-center">Received Documents</h2>
      
      {documents.length === 0 ? (
        <Alert variant="info" className="my-3 text-center">
          You have not received any documents yet.
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
              {documents.map((doc) => {
                console.log("Processing document:", doc);
                return (
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
                      doc.status.includes('ERROR') ? 'bg-secondary' :
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
                      
                      {doc.status === 'PENDING' && !doc.status.includes('ERROR') && userRole !== 'student' && (
                        <>
                          <OverlayTrigger placement="top" overlay={<Tooltip id={`approve-tooltip-${doc.doc_id}`}>Approve Document</Tooltip>}>
                            <Button variant="outline-success" size="sm" className="me-2" onClick={() => handleApproval(doc.doc_id, 'APPROVED')} disabled={refreshing}>
                              <FontAwesomeIcon icon={faCheck} />
                            </Button>
                          </OverlayTrigger>
                          
                          <OverlayTrigger placement="top" overlay={<Tooltip id={`reject-tooltip-${doc.doc_id}`}>Reject Document</Tooltip>}>
                            <Button variant="outline-danger" size="sm" onClick={() => handleApproval(doc.doc_id, 'REJECTED')} disabled={refreshing}>
                              <FontAwesomeIcon icon={faTimes} />
                            </Button>
                          </OverlayTrigger>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
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

export default ReceivedDocuments;
