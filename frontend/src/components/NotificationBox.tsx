import React, { useState, useEffect } from 'react';
import { Dropdown, Badge, Spinner } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faBell } from '@fortawesome/free-solid-svg-icons';
import { Link, useNavigate } from 'react-router-dom';
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
  api_type: 'receiver' | 'sender';
  student_name?: string;
  student_id?: string;
}

interface NotificationBoxProps {
  notificationTrigger: number;
  loggedInUser: string | null;
}

const NotificationBox: React.FC<NotificationBoxProps> = ({ notificationTrigger, loggedInUser }) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  const fetchNotifications = async () => {
    setLoading(true);
    try {
      const user = JSON.parse(localStorage.getItem('user') || '{}');
      const username = user.username || user.id;
      const userEmail = user.email; // Assuming user object has email

      const [receiverNotifications, senderNotifications, unreadReceiver, unreadSender, unreadTraining] = await Promise.all([
        apiFetch(`/api/notifications?username=${encodeURIComponent(username)}`),
        apiFetch(`/api/sender/notifications?username=${encodeURIComponent(username)}`),
        apiFetch(`/api/notifications/unread/count?username=${encodeURIComponent(username)}`),
        apiFetch(`/api/sender/notifications/unread/count?username=${encodeURIComponent(username)}`),
        apiFetch(`/api/training-applications/unread/count?email=${encodeURIComponent(userEmail)}`)
      ]);

      const receiverWithTag = receiverNotifications.map((n: any) => ({ ...n, api_type: 'receiver' as const }));
      const senderWithTag = senderNotifications.map((n: any) => ({
        ...n,
        type: n.notification_type,
        api_type: 'sender' as const,
      }));

      const allNotifications = [...receiverWithTag, ...senderWithTag].sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
      
      setNotifications(allNotifications);
      setUnreadCount(unreadReceiver.count + unreadSender.count + unreadTraining.count);
    } catch (err: any) {
      console.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleNotificationClick = async (notification: Notification) => {
    try {
      const endpoint = notification.api_type === 'receiver'
        ? `/api/notifications/${notification.id}/view`
        : notification.type === 'training_application'
          ? `/api/training-notifications/${notification.id}/view`
          : `/api/sender/notifications/${notification.id}/view`;

      await apiFetch(endpoint, { method: 'POST' });
      fetchNotifications();
      if (notification.type === 'training_application') {
        navigate('/training'); // Navigate to training page for training applications
      } else {
        navigate(`/view/${notification.doc_id}`);
      }
    } catch (error) {
      console.error('Failed to mark notification as read:', error);
    }
  };

  useEffect(() => {
    fetchNotifications();
    const interval = setInterval(fetchNotifications, 5000); // Poll every 5 seconds

    const handleRefresh = () => fetchNotifications();
    window.addEventListener('refreshNotifications', handleRefresh);

    return () => {
      clearInterval(interval); // Cleanup on unmount
      window.removeEventListener('refreshNotifications', handleRefresh);
    };
  }, [notificationTrigger]);

  const renderPotentiallyNestedString = (value: any): string => {
    if (value && typeof value === 'object' && 'String' in value) {
      return value.String;
    }
    return value || '';
  };

  return (
    <Dropdown align="end">
      <Dropdown.Toggle variant="link" id="dropdown-notifications" className="text-decoration-none">
        <FontAwesomeIcon icon={faBell} />
        {unreadCount > 0 && <Badge bg="danger" pill style={{ position: 'absolute', top: '-5px', right: '-5px' }}>{unreadCount}</Badge>}
      </Dropdown.Toggle>

      <Dropdown.Menu style={{ minWidth: '350px' }}>
        <Dropdown.Header>Notifications</Dropdown.Header>
        {loading ? (
          <div className="text-center p-3">
            <Spinner animation="border" size="sm" />
            <p className="mb-0 mt-2">Loading...</p>
          </div>
        ) : notifications.length === 0 ? (
          <Dropdown.ItemText>You have no new notifications.</Dropdown.ItemText>
        ) : (
          notifications.map((notification) => (
            <Dropdown.Item key={notification.id} onClick={() => handleNotificationClick(notification)} className={!notification.viewed ? 'fw-bold' : ''}>
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
                        You have <span className={notification.status === 'APPROVED' ? 'text-success' : 'text-danger'}>{notification.status.toLowerCase()}</span> <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong>.
                      </>
                    ) : (
                      <>
                        <strong>{renderPotentiallyNestedString(notification.approver_username)}</strong> has <span className={notification.status === 'APPROVED' ? 'text-success' : notification.status === 'REJECTED' ? 'text-danger' : ''}>{notification.status.toLowerCase()}</span> <strong>{renderPotentiallyNestedString(notification.doc_name)}</strong>.
                      </>
                    )}
                  </>
                )}
              </div>
              <small className="text-muted">{new Date(notification.created_at).toLocaleString()}</small>
            </Dropdown.Item>
          ))
        )}
        <Dropdown.Divider />
        <Dropdown.Item as={Link} to="/notifications" className="text-center">
          View all notifications
        </Dropdown.Item>
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default NotificationBox;
