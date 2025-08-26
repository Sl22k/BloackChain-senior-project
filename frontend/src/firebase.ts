import { initializeApp } from 'firebase/app';
import { getMessaging, getToken, onMessage } from 'firebase/messaging';

// Your web app's Firebase configuration
// Replace with your actual Firebase config
const firebaseConfig = {
  apiKey: "AIzaSyB4IrFOvJYnPaInNBb2_aEyWO-RaJ83ToA",
  authDomain: "document-approval-system-37e98.firebaseapp.com",
  projectId: "document-approval-system-37e98",
  storageBucket: "document-approval-system-37e98.firebasestorage.app",
  messagingSenderId: "863162130919",
  appId: "1:863162130919:web:98a5b78cf754042cc4dd42",
  measurementId: "G-76CTC82WGM"
};
const app = initializeApp(firebaseConfig);
const messaging = getMessaging(app);

export const getFirebaseToken = async () => {
  try {
    const currentToken = await getToken(messaging, { vapidKey: 'BIj5c8uF14IOZX3rxmwmKkdXYHhrUGSXFh5Fi2Xs-W-NK7c42Cb3mtCzuctAdbLQiUl7dspUIKufW7KTsym-UGk' }); // Replace with your actual VAPID key
    if (currentToken) {
      console.log('Current FCM token:', currentToken);
      return currentToken;
    } else {
      console.log('No FCM token available. Requesting permission...');
      // Request permission to generate a new token
      const permission = await Notification.requestPermission();
      if (permission === 'granted') {
        const newToken = await getToken(messaging, { vapidKey: 'BIj5c8uF14IOZX3rxmwmKkdXYHhrUGSXFh5Fi2Xs-W-NK7c42Cb3mtCzuctAdbLQiUl7dspUIKufW7KTsym-UGk' }); // Replace with your actual VAPID key
        console.log('New FCM token:', newToken);
        return newToken;
      } else {
        console.warn('Notification permission denied.');
        return null;
      }
    }
  } catch (err) {
    console.error('An error occurred while retrieving token:', err);
    return null;
  }
};

export const onMessageListener = (callback: (payload: any) => void) => {
  const unsubscribe = onMessage(messaging, (payload) => {
    console.log('Message received. ', payload);
    callback(payload);
  });
  return unsubscribe;
};
