import React, { useState, useEffect, useCallback } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Table, Button, Spinner, Alert, Badge } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {faEye} from '@fortawesome/free-solid-svg-icons';
import OverlayTrigger from 'react-bootstrap/OverlayTrigger';
import Tooltip from 'react-bootstrap/Tooltip';
import { apiFetch } from '../utils/api';

interface Document {
  doc_id: string;
  doc_name: string;
  uploader_username: string;
  approved_count: number;
  rejected_count: number;
  pending_count: number;
  approvals_map?: { [key: string]: string }; // Changed to approvals_map
  hash?: string;
}

const UploadedDocuments = () => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const location = useLocation();

  const getStatusBadge = (status: string) => {
    switch (status?.toUpperCase()) {
      case 'APPROVED':
        return <Badge bg="success">Approved</Badge>;
      case 'PENDING':
        return <Badge bg="warning" text="dark">Pending</Badge>;
      case 'REJECTED':
        return <Badge bg="danger" text="dark">Rejected</Badge>;
      default:
        return <Badge bg="secondary">{status || 'Unknown'}</Badge>;
    }
  };

  const fetchDocuments = useCallback(async (isLatest = false) => {
    setLoading(true);
    setError(null);
    
    try {
      const user = JSON.parse(localStorage.getItem('user') || '{}');
      const endpoint = isLatest ? '/api/documents/uploaded/latest' : '/api/documents/uploaded';
      const fetchUrl = `${endpoint}?username=${user.id || user.username}`;
      const data = await apiFetch(fetchUrl);
      console.log('Data received from backend for uploaded documents:', data);
      setDocuments(Array.isArray(data) ? data : []);
      
    } catch (err: any) {
      console.error("Fetch error:", err);
      if (err.status === 404) {
        setDocuments([]);
      } else {
        setError(err.message);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const isLatest = location.state?.from === 'upload' || location.state?.from === 'facultyDashboard' || location.state?.from === 'approval';
    fetchDocuments(isLatest);
  }, [fetchDocuments, location.state]);

  if (error) {
    return (
      <Alert variant="danger" className="my-3">
        <Alert.Heading>Error loading documents</Alert.Heading>
        <p>{error}</p>
        <Button variant="primary" onClick={() => fetchDocuments()}>
          Retry
        </Button>
      </Alert>
    );
  }

  if (loading) {
    return (
      <div className="text-center my-5">
        <Spinner animation="border" role="status">
          <span className="visually-hidden">Loading...</span>
        </Spinner>
      </div>
    );
  }

  console.log("documents", documents);
  console.log("loading", loading);

  return (
    <div className="container mt-4">
      <h2 className="mb-4 text-primary text-center">Your Uploaded Documents</h2>
      
      {documents.length === 0 ? (
        <>
          {console.log("Displaying 'no documents' message")}
          <Alert variant="info" className="my-3 text-center">
            there are no uploaded documents yet
          </Alert>
        </>
      ) : (
        <div className="table-responsive">
          <Table striped bordered hover className="shadow-sm">
            <thead className="bg-primary text-white">
              <tr>
                <th>Document Name</th>
                <th>Approver Status</th>
                <th>Approved</th>
                <th>Rejected</th>
                <th>Pending</th>
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
                        overlay={<Tooltip id={`hash-tooltip-${doc.doc_id}`}>{doc.hash}</Tooltip>}
                      >
                        <small className="text-muted">{`Hash: ${doc.hash}`}</small>
                      </OverlayTrigger>
                    )}
                  </td>
                   <td>
                    {doc.approvals_map && Object.keys(doc.approvals_map).length > 0 ? (
                      Object.entries(doc.approvals_map).map(([receiver_username, status]) => (
                        <div key={receiver_username} className="mb-1">
                          <strong>{receiver_username}:</strong> {getStatusBadge(status)} 
                        </div>
                      ))
                    ) : (
                      <div>{getStatusBadge('PENDING')}</div>
                    )}
                  </td>
                  <td><Badge bg="success">{doc.approved_count}</Badge></td>
                  <td><Badge bg="danger">{doc.rejected_count}</Badge></td>
                  <td><Badge bg="warning" text="dark">{doc.pending_count}</Badge></td>
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
        </div>
      )}
    </div>
  );
};

export default UploadedDocuments;