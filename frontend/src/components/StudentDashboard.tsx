import React from 'react';
import { Card, Button, Row, Col } from 'react-bootstrap';
import { useNavigate } from 'react-router-dom'; // Import useNavigate

const StudentDashboard = () => {
  const navigate = useNavigate(); // Initialize useNavigate

  return (
    <div className="dashboard-container">
      <h2 className="my-4 text-center text-primary">Your Dashboard</h2>

      <Row className="g-4">
        <Col md={6}>
          <Card className="h-100 shadow-sm">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary">Training</Card.Title>
              <Card.Text className="flex-grow-1">
                Apply for your training letter.
              </Card.Text>
              <div className="mt-auto">
                <Button variant="primary" onClick={() => navigate('/training')}>Apply for Training</Button>
              </div>
            </Card.Body>
          </Card>
        </Col>
        <Col md={6}>
          <Card className="h-100 shadow-sm">
            <Card.Body className="d-flex flex-column">
              <Card.Title className="text-primary">Received Documents</Card.Title>
              <Card.Text className="flex-grow-1">
                Review and approve documents that have been sent to you for action.
              </Card.Text>
              <div className="mt-auto">
                <Button variant="primary" onClick={() => navigate('/received')}>View Received</Button>
              </div>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default StudentDashboard;
