import React, { useState, useEffect, useCallback } from 'react';
import { BrowserRouter as Router, Route, Routes, useNavigate, useLocation } from 'react-router-dom';
import { Navbar, Container, Nav, Button, Spinner, Alert } from 'react-bootstrap';
import { LinkContainer } from 'react-router-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faUser, faUserPlus, faRightToBracket, faUserMinus } from '@fortawesome/free-solid-svg-icons';
import { onMessageListener } from './firebase';
import { handleApiError, setAuthToken, apiFetch } from './utils/api';
import RoleSelection from './components/RoleSelection';
import Login from './components/Auth/Login';
import Register from './components/Auth/Register';

import StudentDashboard from './components/StudentDashboard';
import FacultyDashboard from './components/FacultyDashboard';
import CoordinatorDashboard from './components/CoordinatorDashboard';
import UploadedDocuments from './components/UploadedDocuments';
import ReceivedDocuments from './components/ReceivedDocuments';
import DocumentViewer from './components/DocumentViewer';
import UploadDocument from './components/UploadDocument';
import NotificationBox from './components/NotificationBox';
import Notifications from './components/Notifications';
import SenderNotifications from './components/SenderNotifications';
import Training from './components/Training';
import AddEditor from './components/AddEditor';
import UpdateApprovers from './components/UpdateApprovers';
import ProtectedRoute from './components/Auth/ProtectedRoute';
import './App.css';

// Define the type for Firebase message payload
type FirebaseMessagePayload = {
  notification?: {
    title: string;
    body: string;
  };
  data?: Record<string, string>;
};

class ErrorBoundary extends React.Component<{children: React.ReactNode}, {hasError: boolean, error?: Error}> {
  constructor(props: {children: React.ReactNode}) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("ErrorBoundary caught:", error, info);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="my-4 text-center">
          <Alert variant="danger">
            Something went wrong: {this.state.error?.message || 'Unknown error'}
          </Alert>
          <Button onClick={() => window.location.reload()}>Refresh Page</Button>
        </div>
      );
    }
    return this.props.children;
  }
}

function MainContent() {
  const navigate = useNavigate();
  const location = useLocation();
  const [backendStatus, setBackendStatus] = useState<'loading' | 'ok' | 'error'>('loading');
  const [statusMessage, setStatusMessage] = useState('');
  const [loggedInUser, setLoggedInUser] = useState<string | null>(null);
  const [userRole, setUserRole] = useState<string | null>(null);
  const [userEmail, setUserEmail] = useState<string | null>(null);
  const [notificationTrigger, setNotificationTrigger] = useState(0);
  const [authLoading, setAuthLoading] = useState(true); // New state for auth loading

  const dashboardPath = () => {
    if (!userRole) return "/";
    switch (userRole) {
        case 'student':
            return '/student-dashboard';
        case 'faculty':
            return '/faculty-dashboard';
        case 'coordinator':
            return '/coordinator-dashboard';
        default:
            return '/';
    }
  };

  const checkBackendStatus = useCallback(async () => {
    try {
      const response = await fetch('/api/status');
      const text = await response.text();
      console.log('Raw backend status response:', text);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}, body: ${text}`);
      }

      const data = JSON.parse(text);
      setBackendStatus('ok');
      setStatusMessage(data.message || 'Backend is running');
    } catch (error: unknown) {
      console.error('Error checking backend status:', error);
      setBackendStatus('error');
      setStatusMessage(handleApiError(error));
    }
  }, []);

  useEffect(() => {
    checkBackendStatus();

    const token = localStorage.getItem('token');
    if (token) {
      setAuthToken(token);
      // Verify token with backend and fetch user data
      const fetchUserData = async () => {
        try {
          console.log('Attempting to verify token with backend. Token:', token);
          const userData = await apiFetch('/api/auth/me', {
            headers: {
              'Authorization': `Bearer ${token}`,
            },
          });
          console.log('User data from /api/auth/me:', userData);
          setLoggedInUser(userData.id);
          setUserRole(userData.role);
          setUserEmail(userData.email);
          console.log('Set loggedInUser:', userData.id, 'and userRole:', userData.role);
        } catch (error) {
          console.error("Failed to verify token or fetch user data:", error);
          localStorage.removeItem('token');
          setAuthToken(null);
          setLoggedInUser(null);
          setUserRole(null);
          navigate('/login'); // Redirect to login if token is invalid or user data fetch fails
        } finally {
          setAuthLoading(false); // Set loading to false after fetch attempt
          console.log('App.tsx: After token verification - loggedInUser:', loggedInUser, 'userRole:', userRole, 'authLoading:', false);
        }
      };
      fetchUserData();
    } else {
      setLoggedInUser(null);
      setUserRole(null);
      setAuthLoading(false); // No token, so auth check is complete
    }
  }, [checkBackendStatus, navigate]); // Removed loggedInUser from dependency array to avoid infinite loops

  useEffect(() => {
    if (!authLoading && userRole && (location.pathname === '/login' || location.pathname === '/')) {
      console.log('App.tsx: Navigating based on userRole:', userRole);
      switch (userRole) {
        case 'student':
          navigate('/student-dashboard');
          break;
        case 'faculty':
          navigate('/faculty-dashboard');
          break;
        case 'coordinator':
          navigate('/coordinator-dashboard');
          break;
        default:
          navigate('/'); // Fallback for unknown roles
      }
    } else if (!authLoading && !userRole && localStorage.getItem('token')) {
      // This case handles scenarios where a token exists but userRole is null after auth check
      // This might indicate an invalid token or a problem with /api/auth/me
      console.log('App.tsx: Token exists but userRole is null after auth check. Redirecting to login.');
      navigate('/login');
    }
  }, [userRole, authLoading, navigate, location.pathname]);

  useEffect(() => {
    if (loggedInUser) {
      const unsubscribe = onMessageListener((payload: FirebaseMessagePayload) => {
        console.log("New notification:", payload);
        setNotificationTrigger(prev => prev + 1);
      });
      return unsubscribe;
    }
  }, [loggedInUser]);

  const handleLogout = () => {
    localStorage.removeItem('token');
    setAuthToken(null); // Clear token from api utility
    setLoggedInUser(null);
    setUserRole(null);
    navigate('/');
  };

  

  return (
    <div className="d-flex flex-column min-vh-100 bg-light">
        <Navbar expand="lg">
          <Container>
            <LinkContainer to={loggedInUser ? dashboardPath() : "/"}>
                <Navbar.Brand>Document Approval System</Navbar.Brand>
            </LinkContainer>
            
            {loggedInUser && (
              <div className="d-flex align-items-center">
                <FontAwesomeIcon icon={faUser} className="ms-3" color="#3B82F6" />
                <Navbar.Text className="ms-2 me-2">
                  Welcome, {String(loggedInUser)} ({userRole})
                </Navbar.Text>
              </div>
            )}
            <Navbar.Toggle aria-controls="basic-navbar-nav" />
            <Navbar.Collapse id="basic-navbar-nav">
              <Nav className="ms-auto">
                {loggedInUser && <div className="me-2"><NotificationBox notificationTrigger={notificationTrigger} loggedInUser={loggedInUser} /></div>}
                {loggedInUser && backendStatus === 'ok' ? (
                  <Button onClick={handleLogout} variant="outline-primary" data-bs-toggle="tooltip" data-bs-placement="bottom" title="Logout">
                    <FontAwesomeIcon icon={faUserMinus} />
                  </Button>
                ) : (
                  <>
                    <LinkContainer to="/login" className="me-2">
                      <Button variant="primary" data-bs-toggle="tooltip" data-bs-placement="bottom" title="Sign In">
                        <FontAwesomeIcon icon={faRightToBracket} />
                      </Button>
                    </LinkContainer>
                    <LinkContainer to="/register">
                      <Button variant="outline-primary" data-bs-toggle="tooltip" data-bs-placement="bottom" title="Register">
                        <FontAwesomeIcon icon={faUserPlus} />
                      </Button>
                    </LinkContainer>
                  </>
                )}
              </Nav>
            </Navbar.Collapse>
          </Container>
        </Navbar>
      <main className="container mt-4 flex-grow-1">
        <ErrorBoundary>
            <Routes>
              <Route path="/" element={<RoleSelection />} />
              <Route path="/login" element={<Login setLoggedInUser={setLoggedInUser} setUserRole={setUserRole} />} />
              <Route path="/register" element={<Register />} />
              {/* Role-specific dashboards */}
              <Route path="/student-dashboard" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student']}><StudentDashboard /></ProtectedRoute>} />
              <Route path="/faculty-dashboard" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['faculty']}><FacultyDashboard /></ProtectedRoute>} />
              <Route path="/coordinator-dashboard" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['coordinator']}><CoordinatorDashboard /></ProtectedRoute>} />
              {/* Other protected routes */}
              <Route path="/received" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student', 'faculty', 'coordinator']}><ReceivedDocuments userRole={userRole} /></ProtectedRoute>} />
              <Route path="/view/:documentId" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student', 'faculty', 'coordinator']}><DocumentViewer userRole={userRole} /></ProtectedRoute>} />
              <Route path="/upload" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student', 'faculty', 'coordinator']}><UploadDocument loggedInUser={loggedInUser} userEmail={userEmail} /></ProtectedRoute>} />
              <Route path="/uploaded" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student', 'faculty', 'coordinator']}><UploadedDocuments /></ProtectedRoute>} />
              <Route path="/notifications" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student', 'faculty', 'coordinator']}><Notifications loggedInUser={loggedInUser} userEmail={userEmail} userRole={userRole} /></ProtectedRoute>} />
              import AddEditor from './components/AddEditor';
import UpdateApprovers from './components/UpdateApprovers';

// ... (rest of the file)

              <Route path="/sender-notifications" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['student', 'faculty', 'coordinator']}><SenderNotifications /></ProtectedRoute>} />
              <Route path="/training" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['coordinator', 'student']}><Training userRole={userRole} userEmail={userEmail} loggedInUser={loggedInUser} /></ProtectedRoute>} />
              <Route path="/documents/:documentId/add-editor" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['faculty', 'coordinator']}><AddEditor /></ProtectedRoute>} />
              <Route path="/documents/:documentId/update-approvers" element={<ProtectedRoute userRole={userRole} authLoading={authLoading} allowedRoles={['faculty', 'coordinator']}><UpdateApprovers /></ProtectedRoute>} />
            </Routes>
          </ErrorBoundary>
      </main>
    </div>
  );
}

function App() {
  return (
    <Router>
      <MainContent />
    </Router>
  );
}

export default App;