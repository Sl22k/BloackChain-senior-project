const express = require('express');
const cors = require('cors');
const fs = require('fs');
const path = require('path');

const app = express();
const port = 3001;

// Middleware
app.use(cors());
app.use(express.json()); // Use express.json() to parse POST request bodies

// Load the coordinator data from the JSON file
const coordinatorsPath = path.join(__dirname, 'coordinators.json');
const coordinators = JSON.parse(fs.readFileSync(coordinatorsPath, 'utf8'));

console.log("Coordinator data loaded:", coordinators);

// --- API Endpoints --- //

/**
 * GET /api/get-coordinator
 * Gets the name of a coordinator for a specific course.
 * Query Params: ?course=COURSE_CODE
 */
app.get('/api/get-coordinator', (req, res) => {
  const { course } = req.query;

  if (!course) {
    return res.status(400).json({ error: 'Course code is required' });
  }

  const coordinatorData = coordinators[course];

  if (coordinatorData) {
    console.log(`Found coordinator for ${course}: ${coordinatorData.name}`);
    res.json({ course, coordinator: coordinatorData }); // Return the whole object
  } else {
    console.log(`Coordinator not found for course: ${course}`);
    res.status(404).json({ course, coordinator: 'Coordinator not found' });
  }
});

/**
 * POST /api/check-coordinator
 * Checks if a given email belongs to any of the course coordinators.
 * Request Body: { "email": "user@example.com" }
 */
app.post('/api/check-coordinator', (req, res) => {
  const { email } = req.body;

  if (!email) {
    return res.status(400).json({ error: 'Email is required' });
  }

  // Look through all the coordinator data to see if the email exists (case-insensitive)
  const isCoordinator = Object.values(coordinators).some(coord => coord.email.toLowerCase() === email.toLowerCase());

  if (isCoordinator) {
    console.log(`Verified coordinator: ${email}`);
    // You could expand this to return which courses they coordinate
    res.json({ isCoordinator: true, email });
  } else {
    console.log(`Failed coordinator check for: ${email}`);
    res.json({ isCoordinator: false, email });
  }
});


app.listen(port, () => {
  console.log(`Manual coordinator server listening at http://localhost:${port}`);
});
