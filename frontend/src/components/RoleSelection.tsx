import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Container, Button, Row, Col, Card } from 'react-bootstrap';

const RoleSelection = () => {
  const navigate = useNavigate();

  const handleRoleSelection = (role: string) => {
    navigate('/login', { state: { role } });
  };

  return (
    <Container className="d-flex align-items-center justify-content-center" style={{ minHeight: '100vh' }}>
      <Card className="p-4 shadow-sm">
        <Card.Body>
          <h2 className="text-center mb-4">Select Your Role</h2>
          <Row>
            <Col className="d-grid gap-2">
              <Button variant="primary" size="lg" onClick={() => handleRoleSelection('student')}>
                Student
              </Button>
            </Col>
            <Col className="d-grid gap-2">
              <Button variant="secondary" size="lg" onClick={() => handleRoleSelection('faculty')}>
                Faculty
              </Button>
            </Col>
          </Row>
        </Card.Body>
      </Card>
    </Container>
  );
};

export default RoleSelection;
