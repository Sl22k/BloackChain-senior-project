import React, { useState, useEffect } from 'react';
import { Card, Button, Row, Col, Badge } from 'react-bootstrap';
import { Link } from 'react-router-dom';
import { apiFetch } from '../utils/api';

interface TrainingApplication {
  id: number;
  studentName: string;
  studentId: string;
  cpr: string;
  nationality: string;
  telephone: string;
  email: string;
  courseCode: string;
  coordinatorEmail: string;
  status: string;
  createdAt: string;
}

const CoordinatorDashboard = () => {
  const [applications, setApplications] = useState<TrainingApplication[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchApplications = async () => {
      try {
        const user = JSON.parse(localStorage.getItem('user') || '{}');
        const coordinatorEmail = user.email;
        if (coordinatorEmail) {
          const data = await apiFetch(`/api/applications?email=${coordinatorEmail}`);
          setApplications(data || []);
        }
      } catch (error) {
        console.error('Failed to fetch applications:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchApplications();
    const interval = setInterval(fetchApplications, 5000); // Poll every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const pendingApplications = applications.filter(app => app.status === 'pending');

  return (
    <div className="dashboard-container">
      <h2 className="my-4 text-center text-primary">Coordinator Dashboard</h2>

      <Row className="g-4">
        <Col md={4}>
          <Card className="h-100 shadow-sm">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary">Uploaded Documents</Card.Title>
              <Card.Text className="flex-grow-1">
                View the status of documents you have uploaded and manage them.
              </Card.Text>
              <div className="mt-auto">
                <Link to="/uploaded" state={{ from: 'coordinatorDashboard' }}>
                  <Button variant="primary">View Uploaded</Button>
                </Link>
              </div>
            </Card.Body>
          </Card>
        </Col>
        <Col md={4}>
          <Card className="h-100 shadow-sm">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary">Received Documents</Card.Title>
              <Card.Text className="flex-grow-1">
                Review and approve documents that have been sent to you for action.
              </Card.Text>
              <div className="mt-auto">
                <Link to="/received">
                  <Button variant="primary">View Received</Button>
                </Link>
              </div>
            </Card.Body>
          </Card>
        </Col>
        <Col md={4}>
          <Card className="h-100 shadow-sm">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary">Upload Document</Card.Title>
              <Card.Text className="flex-grow-1">
                Upload a new document to the system.
              </Card.Text>
              <div className="mt-auto">
                <Link to="/upload">
                  <Button variant="success">Upload New Document</Button>
                </Link>
              </div>
            </Card.Body>
          </Card>
        </Col>
        <Col md={4}>
          <Card className="h-100 shadow-sm bg-light">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary d-flex justify-content-between align-items-start">
                Training Management
                {loading ? (
                  <Badge bg="secondary">Loading...</Badge>
                ) : pendingApplications.length > 0 ? (
                  <Badge bg="danger" pill>
                    {pendingApplications.length} New
                  </Badge>
                ) : null}
              </Card.Title>
              <Card.Text className="flex-grow-1">
                Review student applications and issue training letters.
              </Card.Text>
              <div className="mt-auto">
                <Link to="/training">
                  <Button variant="primary">Go to Training Page</Button>
                </Link>
              </div>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default CoordinatorDashboard;
