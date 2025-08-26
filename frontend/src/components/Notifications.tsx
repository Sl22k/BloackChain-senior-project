import React, { useState, useEffect } from 'react';
import { Alert, Button, Spinner, ListGroup, Badge } from 'react-bootstrap';
import { useNavigate } from 'react-router-dom';
import { apiFetch } from '../utils/api';

interface Notification {
  id: number;
  doc_id: string;
  doc_name: string;
  sender_username?: string;
  approver_username?: string;
  status: string;
  created_at: string;
  type: 'new_document' | 'status_change' | 'training_application' | 'document_approved' | 'document_rejected';
  viewed: boolean;
  student_name?: string;
  student_id?: string;
}

interface NotificationsProps {
  loggedInUser: string | null;
  userEmail: string | null;
  userRole: string | null;
}

const Notifications: React.FC<NotificationsProps> = ({ loggedInUser, userEmail, userRole }) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  const fetchNotifications = async () => {
    setLoading(true);
    setError(null);
    try {
      if (!loggedInUser || !userEmail) {
        setError("User details not found. Please log in again.");
        setLoading(false);
        return;
      }

      const promises = [
        apiFetch(`/api/notifications?username=${encodeURIComponent(loggedInUser)}`),
        apiFetch(`/api/sender/notifications?username=${encodeURIComponent(loggedInUser)}`),
      ];

      if (userRole === 'coordinator') {
        promises.push(apiFetch(`/api/training-applications/notifications?email=${encodeURIComponent(userEmail)}`));
      }

      const [receiverNotifications, senderNotifications, trainingNotifications] = await Promise.all(promises);

      const transformedSenderNotifications = senderNotifications.map((n: any) => ({
        ...n,
        type: n.notification_type,
      }));

      let transformedTrainingNotifications: any[] = [];
      if (trainingNotifications) {
        transformedTrainingNotifications = trainingNotifications.map((n: any) => ({
          id: n.id,
          doc_id: n.id, // Using application ID as doc_id for consistency
          doc_name: `Training Application from ${n.studentName}`,
          sender_username: n.studentName,
          student_id: n.studentId,
          status: n.status,
          created_at: n.createdAt,
          type: 'training_application',
          viewed: n.viewed,
        }));
      }

      const allNotifications = [...receiverNotifications, ...transformedSenderNotifications, ...transformedTrainingNotifications].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());

      setNotifications(allNotifications);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleNotificationClick = async (notification: Notification) => {
    try {
      const isReceiverNotification = !!notification.sender_username;
      const endpoint = isReceiverNotification
        ? `/api/notifications/${notification.id}/view`
        : notification.type === 'training_application'
          ? `/api/training-notifications/${notification.id}/view`
          : `/api/sender/notifications/${notification.id}/view`;

      await apiFetch(endpoint, { method: 'POST' });
      if (notification.type === 'training_application') {
        navigate('/training');
      } else {
        navigate(`/view/${notification.doc_id}`);
      }
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
    }
  };

  useEffect(() => {
    fetchNotifications();
  }, []);

  const renderPotentiallyNestedString = (value: any): string => {
    if (value && typeof value === 'object' && 'String' in value) {
      return value.String;
    }
    return value || '';
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
      <h2 className="mb-4 text-primary text-center">All Notifications</h2>
      {notifications.length === 0 ? (
        <Alert variant="info" className="my-3 text-center">
          You have no notifications.
        </Alert>
      ) : (
        <ListGroup className="shadow-sm">
          {notifications.map((notification) => (
            <ListGroup.Item key={notification.id} action onClick={() => handleNotificationClick(notification)} className={`${!notification.viewed ? 'fw-bold' : 'text-muted'}`}>
              <div>
                {notification.type === 'new_document' ? (
                  <>
                    New document <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong> received from <strong>{renderPotentiallyNestedString(notification.sender_username)}</strong>.
                  </>
                ) : notification.type === 'training_application' ? (
                  <>
                    Student <strong>{renderPotentiallyNestedString(notification.student_name)} ({renderPotentiallyNestedString(notification.student_id)})</strong> has applied for a training letter.
                  </>
                ) : notification.type === 'document_approved' ? (
                  <>
                    <strong>{renderPotentiallyNestedString(notification.approver_username)}</strong> has approved your document <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong>.
                  </>
                ) : notification.type === 'document_rejected' ? (
                  <>
                    <strong>{renderPotentiallyNestedString(notification.approver_username)}</strong> has rejected your document <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong>.
                  </>
                ) : (
                  <>
                    {renderPotentiallyNestedString(notification.approver_username) === loggedInUser ? (
                      <>
                        The status of document <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong> has been updated to <Badge bg={notification.status === 'APPROVED' ? 'success' : 'danger'}>{notification.status}</Badge> by you.
                      </>
                    ) : (
                      <>
                        The status of document <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong> has been updated to <Badge bg={notification.status === 'APPROVED' ? 'success' : 'danger'}>{notification.status}</Badge> by <strong>{renderPotentiallyNestedString(notification.approver_username)}</strong>.
                      </>
                    )}
                  </>
                )}
                <br />
                <small>{new Date(notification.created_at).toLocaleString()}</small>
              </div>
            </ListGroup.Item>
          ))}
        </ListGroup>
      )}
    </div>
  );
}

export default Notifications;