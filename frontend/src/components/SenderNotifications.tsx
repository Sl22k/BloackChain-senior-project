import React, { useState, useEffect } from 'react';
import { Alert, Button, Spinner, ListGroup, Badge } from 'react-bootstrap';
import { apiFetch } from '../utils/api';

interface SenderNotification {
  id: number;
  doc_id: string;
  doc_name: string;
  approver_username: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED' | string;
  created_at: string;
  notification_type: string;
  viewed: boolean;
}

const SenderNotifications = () => {
  const [notifications, setNotifications] = useState<SenderNotification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchNotifications = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const userJson = localStorage.getItem('user');
      if (!userJson) {
        throw new Error('User not found in localStorage');
      }

      const user = JSON.parse(userJson);
      const username = user?.username || user?.id;
      if (!username) {
        throw new Error('Username not available');
      }

      const data = await apiFetch(
        `/api/sender/notifications?username=${encodeURIComponent(username)}`
      );

      setNotifications(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An unknown error occurred');
    } finally {
      setLoading(false);
    }
  };

  const handleView = async (notificationId: number) => {
    try {
      await apiFetch(`/api/sender/notifications/${notificationId}/view`, {
        method: 'POST',
      });
      fetchNotifications();
    } catch (error) {
      console.error('Failed to mark sender notification as read:', error);
    }
  };

  useEffect(() => {
    fetchNotifications();
    
    const interval = setInterval(fetchNotifications, 5000);
    return () => clearInterval(interval);
  }, []);

  const getBadgeVariant = (status: string) => {
    switch (status) {
      case 'APPROVED':
        return 'success';
      case 'REJECTED':
        return 'danger';
      case 'PENDING':
        return 'warning';
      default:
        return 'secondary';
    }
  };

  if (loading) {
    return (
      <div className="text-center my-5">
        <Spinner animation="border" role="status">
          <span className="visually-hidden">Loading notifications...</span>
        </Spinner>
        <p className="mt-2">Loading notifications...</p>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="danger" className="my-3">
        <Alert.Heading>Error loading notifications</Alert.Heading>
        <p>{error}</p>
        <Button variant="primary" onClick={fetchNotifications}>
          Try Again
        </Button>
      </Alert>
    );
  }

  return (
    <div className="container mt-4">
      <h2 className="mb-4">Sender Notifications</h2>
      {notifications.length === 0 ? (
        <Alert variant="info" className="my-3">
          You have no new notifications.
        </Alert>
      ) : (
        <ListGroup>
          {notifications.map((notification) => (
            <ListGroup.Item key={notification.id} className={`mb-2 ${notification.viewed ? 'text-muted' : ''}`}>
              <div className="d-flex justify-content-between align-items-start">
                <div>
                  {notification.notification_type === 'status_change' && (
                    <>
                      <strong>Document:</strong> {notification.doc_name}
                      <br />
                      <strong>Receiver:</strong> {notification.approver_username}
                      <br />
                      <strong>Status:</strong>{' '}
                      <Badge bg={getBadgeVariant(notification.status)}>
                        {notification.status}
                      </Badge>
                    </>
                  )}
                </div>
                <small className="text-muted">
                  {new Date(notification.created_at).toLocaleString()}
                </small>
                {!notification.viewed && (
                  <Button variant="outline-primary" size="sm" onClick={() => handleView(notification.id)}>
                    Mark as Read
                  </Button>
                )}
              </div>
            </ListGroup.Item>
          ))}
        </ListGroup>
      )}
    </div>
  );
};

export default SenderNotifications;