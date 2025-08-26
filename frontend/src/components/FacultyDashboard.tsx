import React from 'react';
import { Card, Button, Row, Col } from 'react-bootstrap';
import { Link } from 'react-router-dom';

const FacultyDashboard = () => {
  return (
    <div className="dashboard-container">
      <h2 className="my-4 text-center text-primary">Your Dashboard</h2>

      <Row className="g-4">
        <Col md={4}>
          <Card className="h-100 shadow-sm">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary">Uploaded Documents</Card.Title>
              <Card.Text className="flex-grow-1">
                View the status of documents you have uploaded and manage them.
              </Card.Text>
              <div className="mt-auto">
                <Link to="/uploaded" state={{ from: 'facultyDashboard' }}>
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
      </Row>
    </div>
  );
};

export default FacultyDashboard;
