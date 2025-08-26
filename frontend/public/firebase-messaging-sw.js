importScripts('https://www.gstatic.com/firebasejs/9.0.0/firebase-app-compat.js');
importScripts('https://www.gstatic.com/firebasejs/9.0.0/firebase-messaging-compat.js');

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

// Initialize the Firebase app in the service worker by reusing the config from the main app
firebase.initializeApp(firebaseConfig);

const messaging = firebase.messaging();

// Handle background messages
messaging.onBackgroundMessage((payload) => {
  console.log('[firebase-messaging-sw.js] Received background message ', payload);

  const notificationTitle = payload.notification.title;
  const notificationOptions = {
    body: payload.notification.body,
    icon: '/logo192.png', // You can customize this icon
    data: payload.data, // Pass data payload to the notification
  };

  self.registration.showNotification(notificationTitle, notificationOptions);
});
