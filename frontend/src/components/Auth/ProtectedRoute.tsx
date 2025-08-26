import React from 'react';
import { Navigate } from 'react-router-dom';
import { Spinner } from 'react-bootstrap'; // Import Spinner for loading indicator

interface ProtectedRouteProps {
  children: React.ReactElement;
  userRole: string | null; // Prop for user's role
  authLoading: boolean;    // Prop for authentication loading status
  allowedRoles: string[]; // New prop for allowed roles
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ children, userRole, authLoading, allowedRoles }) => {
  console.log('ProtectedRoute: Received props - userRole:', userRole, 'authLoading:', authLoading, 'allowedRoles:', allowedRoles);
  
  if (authLoading) {
    return (
      <div className="my-4 text-center">
        <Spinner animation="border" variant="primary" />
        <p className="mt-2 text-secondary">Loading user data...</p>
      </div>
    );
  }

  if (!userRole) {
    console.log('ProtectedRoute: No user role, redirecting to /login');
    return <Navigate to="/login" replace />;
  }

  if (!allowedRoles.includes(userRole)) {
    console.log(`ProtectedRoute: Role '${userRole}' not allowed. Redirecting to dashboard.`);
    // Redirect to a generic dashboard or a "not authorized" page
    // For now, let's redirect to the user's dashboard
    switch (userRole) {
      case 'student':
        return <Navigate to="/student-dashboard" replace />;
      case 'faculty':
        return <Navigate to="/faculty-dashboard" replace />;
      case 'coordinator':
        return <Navigate to="/coordinator-dashboard" replace />;
      default:
        return <Navigate to="/" replace />;
    }
  }

  console.log('ProtectedRoute: Access granted for role:', userRole);
  return children;
};

export default ProtectedRoute;