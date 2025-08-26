import React, { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Form, Button, Container, Row, Col, Card, Alert, Spinner } from 'react-bootstrap';
import { apiFetch, handleApiError, setAuthToken } from '../../utils/api';
import { getFirebaseToken } from '../../firebase';

interface LoginProps {
  setLoggedInUser: (user: string | null) => void;
  setUserRole: (role: string | null) => void;
}

const Login = ({ setLoggedInUser, setUserRole }: LoginProps) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const role = location.state?.role;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const fcmToken = await getFirebaseToken();
      // Send the email value in the `username` field to match the backend API
      const response = await apiFetch('/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username: email, password, fcm_token: fcmToken }),
      });

      // This is the user object from your main backend
      const user = response.user;
      console.log('Logged in user object (before coordinator check):', user); // Add this line to inspect the user object

      // If the user is a faculty, check if they are a coordinator
      if (user.role === 'faculty') {
        try {
          const checkCoordinatorResponse = await fetch('http://localhost:3001/api/check-coordinator', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            // IMPORTANT: Your /api/auth/login response must include the user's email
            body: JSON.stringify({ email: user.email }),
          });
          const coordinatorData = await checkCoordinatorResponse.json();
          console.log('Coordinator check response (coordinatorData.isCoordinator):', coordinatorData.isCoordinator);

          if (coordinatorData.isCoordinator) {
            console.log('User is a coordinator, upgrading role.');
            user.role = 'coordinator'; // Upgrade the role
            console.log('User object after role upgrade:', user);
          }
        } catch (e) {
          console.error("Could not connect to coordinator check service. Proceeding as regular faculty.", e);
          // Non-critical error, so we don't block login. The user can proceed as a regular faculty member.
        }
      }

      localStorage.setItem('token', response.token);
      setAuthToken(response.token); // Set token in api utility

      console.log('Calling setLoggedInUser and setUserRole with:', user.id, user.role);
      setLoggedInUser(user.id);
      setUserRole(user.role);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container className="d-flex align-items-center justify-content-center" style={{ minHeight: '80vh' }}>
      <Row className="justify-content-center w-100">
        <Col md={6}>
          <Card className="p-4 shadow-sm">
            <Card.Body>
              <h2 className="text-center mb-4">Login as {role}</h2>
              {error && <Alert variant="danger">{error}</Alert>}
              <Form onSubmit={handleSubmit}>
                <Form.Group className="mb-3" controlId="email">
                  <Form.Label>Email Address</Form.Label>
                  <Form.Control
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    disabled={loading}
                  />
                </Form.Group>

                <Form.Group className="mb-3" controlId="password">
                  <Form.Label>Password</Form.Label>
                  <Form.Control
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    disabled={loading}
                  />
                </Form.Group>

                <Button variant="primary" type="submit" className="w-100" disabled={loading}>
                  {loading ? <Spinner animation="border" size="sm" /> : 'Login'}
                </Button>
              </Form>
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
};

export default Login;