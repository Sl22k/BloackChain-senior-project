import React, { useState, useEffect } from 'react';
import { Container, Row, Col, Card, Button, Form, Spinner, Alert, Table, Modal } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faEye, faTrashCan, faFilePen, faShare } from '@fortawesome/free-solid-svg-icons';
import { apiFetch } from '../utils/api';
import { color } from 'html2canvas/dist/types/css/types/color';
import jsPDF from 'jspdf';
import html2canvas from 'html2canvas';

// Define the shape of the coordinator object for type safety
interface Coordinator {
  name: string;
  email: string;
}

// Define the shape of a Training Application
interface TrainingApplication {
  id: number;
  studentName: string;
  studentId: string;
  cpr: string;
  nationality: string;
  telephone: string;
  email: string;
  courseCode: string;
  coordinatorEmail: string;
  status: string;
  createdAt: string; // Assuming string format from backend
}

interface TrainingProps {
  userRole: string | null;
  userEmail: string | null;
  loggedInUser: string | null;
}

const Training: React.FC<TrainingProps> = ({ userRole, userEmail, loggedInUser }) => {
  // --- STATE MANAGEMENT ---
  const [apply, setApply] = useState<boolean | null>(null);
  const [department, setDepartment] = useState('');
  const [course, setCourse] = useState('');
  const [coordinator, setCoordinator] = useState<Coordinator | null>(null);
  const [isLoadingCoordinator, setIsLoadingCoordinator] = useState(false);

  // State for the student information form
  const [studentInfo, setStudentInfo] = useState({
    studentName: '',
    studentId: '',
    cpr: '',
    nationality: '',
    telephone: '',
    email: '',
  });

  // State for submission handling (student view)
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitMessage, setSubmitMessage] = useState('');
  const [submitError, setSubmitError] = useState('');
  const [alreadyAppliedMessage, setAlreadyAppliedMessage] = useState('');
  const [proceedMessage, setProceedMessage] = useState('');

  // State for fetching applications (coordinator view)
  const [applications, setApplications] = useState<TrainingApplication[]>([]);
  const [isLoadingApplications, setIsLoadingApplications] = useState(true);
  const [applicationsError, setApplicationsError] = useState<string | null>(null);

  // State for the details modal (coordinator view)
  const [showModal, setShowModal] = useState(false);
  const [selectedApp, setSelectedApp] = useState<TrainingApplication | null>(null);
  const [showEditModal, setShowEditModal] = useState(false);
  const [selectedAppForEdit, setSelectedAppForEdit] = useState<TrainingApplication | null>(null);

  // --- EFFECTS ---

  // Effect for fetching coordinator details (student view)
  useEffect(() => {
    if (course && userRole === 'student') {
      setIsLoadingCoordinator(true);
      setCoordinator(null);
      fetch(`http://localhost:3001/api/get-coordinator?course=${course}`)
        .then(res => res.json())
        .then(data => {
          if (data.coordinator && typeof data.coordinator === 'object') {
            setCoordinator(data.coordinator);
          } else {
            setCoordinator({ name: 'Not Found', email: '' });
          }
        })
        .catch(() => setCoordinator({ name: 'Error fetching data', email: '' }))
        .finally(() => setIsLoadingCoordinator(false));
    }
  }, [course, userRole]);

  // Effect for fetching applications (coordinator view)
  useEffect(() => {
    if (userRole === 'coordinator' || userRole === 'faculty') {
      if (userEmail) {
        const coordinatorEmail = userEmail;

        if (coordinatorEmail) {
          setIsLoadingApplications(true);
          setApplicationsError(null);
          apiFetch(`/api/applications?email=${coordinatorEmail}`)
            .then(data => {
              setApplications(data || []);
            })
            .catch(error => {
              setApplicationsError(error.message || 'Failed to load applications.');
            })
            .finally(() => {
              setIsLoadingApplications(false);
            });
        } else {
          setApplicationsError('Coordinator email not found in user data.');
          setIsLoadingApplications(false);
        }
      } else {
        setApplicationsError('User data not found.');
        setIsLoadingApplications(false);
      }
    }
  }, [userRole, userEmail]);

  // --- HANDLERS ---
  const handleStudentInfoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setStudentInfo(prev => ({ ...prev, [name]: value }));
  };

  const handleStudentIdBlur = () => {
    const { studentId } = studentInfo;
    if (!studentId) return;

    // Clear previous messages
    setAlreadyAppliedMessage('');
    setProceedMessage('');
    setSubmitError('');

    apiFetch(`/api/applications/student/${encodeURIComponent(studentId)}`)
      .then(data => {
        if (data) {
          setAlreadyAppliedMessage('You have already applied for the letter' );
        }
      })
      .catch(error => {
        if (error.message && error.message.includes('404')) {
          setProceedMessage('This Student ID is valid to apply.');
        } else {
          setSubmitError(`Error checking Student ID: ${error.message}`);
        }
      });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault(); // Prevent default form submission
    setSubmitMessage('');
    setSubmitError('');
    setAlreadyAppliedMessage(''); // Clear any previous messages

    if (!coordinator || !coordinator.email) {
      setSubmitError('Cannot submit: Coordinator email is not available.');
      return;
    }
    setIsSubmitting(true);

    const applicationData = { ...studentInfo, courseCode: course, coordinatorEmail: coordinator.email };

    try {
      const result = await apiFetch('/api/applications', { 
        method: 'POST', 
        body: JSON.stringify(applicationData) 
      });
      setSubmitMessage(result.message || 'Application submitted successfully!');
    } catch (error: any) {
      setSubmitError(error.message || 'An error occurred during submission.');
    }
  };

  const handleViewDetails = (app: TrainingApplication) => {
    setSelectedApp(app);
    setShowModal(true);
  };

  const handleEdit = (app: TrainingApplication) => {
    setSelectedAppForEdit(app);
    setShowEditModal(true);
  };

  // --- DATA ---
  const departmentCourses: { [key: string]: string[] } = {
    CS: ['ITCC481', 'ITCS481', 'ITSE481'],
    CE: ['ITAD299', 'ITCE490', 'ITNE400'],
    IS: ['ITIS483', 'ITCY482'],
  };

  // --- RENDER LOGIC ---
  const renderStudentInfoForm = () => (
    <div className="mt-4 pt-3 border-top">
      <h4 className="text-secondary mb-3">Student Information</h4>
      <Row>
        <Col md={6}><Form.Group className="mb-3"><Form.Label>Full Name</Form.Label><Form.Control name="studentName" type="text" value={studentInfo.studentName} onChange={handleStudentInfoChange} required disabled={!!alreadyAppliedMessage} /></Form.Group></Col>
        <Col md={6}>
          <Form.Group className="mb-3">
            <Form.Label>Student ID</Form.Label>
            <Form.Control 
              name="studentId" 
              type="text" 
              value={studentInfo.studentId} 
              onChange={handleStudentInfoChange} 
              onBlur={handleStudentIdBlur} // Check on blur
              required 
              disabled={!!alreadyAppliedMessage} 
            />
          </Form.Group>
        </Col>
      </Row>
      <Row>
        <Col md={6}><Form.Group className="mb-3"><Form.Label>CPR Number</Form.Label><Form.Control name="cpr" type="text" value={studentInfo.cpr} onChange={handleStudentInfoChange} required disabled={!!alreadyAppliedMessage} /></Form.Group></Col>
        <Col md={6}><Form.Group className="mb-3"><Form.Label>Nationality</Form.Label><Form.Control name="nationality" type="text" value={studentInfo.nationality} onChange={handleStudentInfoChange} required disabled={!!alreadyAppliedMessage} /></Form.Group></Col>
      </Row>
      <Row>
        <Col md={6}><Form.Group className="mb-3"><Form.Label>Telephone</Form.Label><Form.Control name="telephone" type="tel" value={studentInfo.telephone} onChange={handleStudentInfoChange} required disabled={!!alreadyAppliedMessage} /></Form.Group></Col>
        <Col md={6}><Form.Group className="mb-3"><Form.Label>Email Address</Form.Label><Form.Control name="email" type="email" value={studentInfo.email} onChange={handleStudentInfoChange} required disabled={!!alreadyAppliedMessage} /></Form.Group></Col>
      </Row>
    </div>
  );

  const renderStudentView = () => {
    if (submitMessage) {
      return <Alert variant="success">{submitMessage}</Alert>;
    }

    return (
      <Form onSubmit={handleSubmit}>
        <Form.Group className="mb-3">
          <Form.Label>Have you completed 85 course hours?</Form.Label>
          <div>
            <Form.Check type="radio" label="Yes" name="apply" value="yes" onChange={() => setApply(true)} inline />
            <Form.Check type="radio" label="No" name="apply" value="no" onChange={() => setApply(false)} inline />
          </div>
        </Form.Group>

        {apply === false && (<Alert variant="warning">You are not eligible to apply for training.</Alert>)}

        {apply && (
          <>
            <Form.Group className="mb-3">
              <Form.Label>Select your department:</Form.Label>
              <Form.Select value={department} onChange={(e) => { setDepartment(e.target.value); setCourse(''); }} disabled={!!alreadyAppliedMessage}>
                <option value="">-- Select --</option>
                <option value="CS">CS</option>
                <option value="CE">CE</option>
                <option value="IS">IS</option>
              </Form.Select>
            </Form.Group>
            {department && (
              <Form.Group className="mb-3">
                <Form.Label>Select your course:</Form.Label>
                <Form.Select value={course} onChange={(e) => setCourse(e.target.value)} disabled={!!alreadyAppliedMessage}>
                  <option value="">-- Select --</option>
                  {departmentCourses[department]?.map(c => (<option key={c} value={c}>{c}</option>))}
                </Form.Select>
              </Form.Group>
            )}
          </>
        )}
        
        {isLoadingCoordinator && <p className="text-info">Loading coordinator...</p>}
        
        {coordinator && !isLoadingCoordinator && (
          <div className="mt-4 p-3 bg-light rounded">
            <h5 className="text-primary">Course Coordinator</h5>
            <p className="mb-0">{coordinator.name}</p>
          </div>
        )}
        
        {course && !isLoadingCoordinator && renderStudentInfoForm()}

        {/* Display messages based on the check */}
        {alreadyAppliedMessage && <Alert variant="danger" className="mt-3">{alreadyAppliedMessage}</Alert>}
        {proceedMessage && <Alert variant="success" className="mt-3">{proceedMessage}</Alert>}
        {submitError && <Alert variant="danger" className="mt-3">{submitError}</Alert>}
        
        {course && !isLoadingCoordinator && (
          <Button variant="primary" type="submit" className="mt-3 w-100" disabled={isSubmitting || !!alreadyAppliedMessage}>
            {isSubmitting ? <><Spinner as="span" animation="border" size="sm" /> Submitting...</> : 'Submit Application'}
          </Button>
        )}
      </Form>
    );
  };

  const handleDelete = async (appId: number) => {
    console.log("Delete button clicked for application ID:", appId);
    if (window.confirm('Are you sure you want to delete this application?')) {
      console.log("Confirmation received for ID:", appId);
      try {
        console.log("Attempting apiFetch for ID:", appId);
        const response = await apiFetch(`/api/applications/${appId}`, { method: 'DELETE' });
        console.log("apiFetch completed for ID:", appId, "Response:", response);
        setApplications(prevApps => prevApps.filter(app => app.id !== appId));
        alert('Application deleted successfully!');
      } catch (error: any) {
        console.error("Error during apiFetch or deletion for ID:", appId, error);
        alert(`Failed to delete application: ${error.message}`);
      }
    }
  };

  const renderFacultyView = () => {
    if (isLoadingApplications) {
      return (<div className="text-center"><Spinner animation="border" /><p>Loading applications...</p></div>);
    }
    if (applicationsError) {
      return (<Alert variant="danger">Error: {applicationsError}</Alert>);
    }
    if (applications.length === 0) {
      return (<Alert variant="info">No training applications submitted to you yet.</Alert>);
    }

    return (
      <div className="mt-4">
        <h4 className="text-primary mb-3">Submitted Training Applications</h4>
        <Table striped bordered hover responsive>
          <thead>
            <tr>
              <th>ID</th>
              <th>Student Name</th>
              <th>Student ID</th>
              <th>Course</th>
              <th>Status</th>
              <th>Submitted On</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {applications.map(app => (
              <tr key={app.id}>
                <td>{app.id}</td>
                <td>{app.studentName}</td>
                <td>{app.studentId}</td>
                <td>{app.courseCode}</td>
                <td>{app.status}</td>
                <td>{new Date(app.createdAt).toLocaleDateString()}</td>
                <td>
                  <Button variant="info" size="sm" className="me-2" onClick={() => handleViewDetails(app)} title="View Details"><FontAwesomeIcon icon={faEye} /></Button>
                  <Button variant="warning" size="sm" className="me-2" onClick={() => handleEdit(app)} title="Edit Application"><FontAwesomeIcon icon={faFilePen} /></Button>
                  <Button variant="danger" size="sm" onClick={() => handleDelete(app.id)} title="Delete Application"><FontAwesomeIcon icon={faTrashCan} /></Button>
                </td>
              </tr>
            ))}
          </tbody>
        </Table>

        {selectedApp && (
          <Modal show={showModal} onHide={() => setShowModal(false)} size="lg">
            <Modal.Header closeButton>
              <Modal.Title>Application Details - {selectedApp.studentName}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
              <Form>
                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Student Name</Form.Label>
                      <Form.Control type="text" readOnly defaultValue={selectedApp.studentName} />
                    </Form.Group>
                  </Col>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Student ID</Form.Label>
                      <Form.Control type="text" readOnly defaultValue={selectedApp.studentId} />
                    </Form.Group>
                  </Col>
                </Row>
                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>CPR</Form.Label>
                      <Form.Control type="text" readOnly defaultValue={selectedApp.cpr} />
                    </Form.Group>
                  </Col>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Nationality</Form.Label>
                      <Form.Control type="text" readOnly defaultValue={selectedApp.nationality} />
                    </Form.Group>
                  </Col>
                </Row>
                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Telephone</Form.Label>
                      <Form.Control type="text" readOnly defaultValue={selectedApp.telephone} />
                    </Form.Group>
                  </Col>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Email</Form.Label>
                      <Form.Control type="email" readOnly defaultValue={selectedApp.email} />
                    </Form.Group>
                  </Col>
                </Row>
                <Row>
                  <Col md={6}>
                    <Form.Group className="mb-3">
                      <Form.Label>Course Code</Form.Label>
                      <Form.Control type="text" readOnly defaultValue={selectedApp.courseCode} />
                    </Form.Group>
                  </Col>
                </Row>
                 <Form.Group className="mb-3">
                    <Form.Label>Submitted At</Form.Label>
                    <Form.Control type="text" readOnly defaultValue={new Date(selectedApp.createdAt).toLocaleDateString()} />
                </Form.Group>
              </Form>
            </Modal.Body>
            <Modal.Footer>
              <Button variant="secondary" onClick={() => setShowModal(false)}>
                Close
              </Button>
            </Modal.Footer>
          </Modal>
        )}

        {selectedAppForEdit && (
          <EditTrainingLetterModal
            show={showEditModal}
            onHide={() => setShowEditModal(false)}
            application={selectedAppForEdit}
            coordinatorUsername={loggedInUser}
          />
        )}
      </div>
    );
  };

const EditTrainingLetterModal = ({ show, onHide, application, coordinatorUsername }: {
  show: boolean;
  onHide: () => void;
  application: TrainingApplication;
  coordinatorUsername: string | null;
}) => {
  const [editableFields, setEditableFields] = useState({
    date: new Date().toLocaleDateString(),
    studentName: application.studentName,
    studentId: application.studentId,
    cpr: application.cpr,
    nationality: application.nationality,
    telephone: application.telephone,
    email: application.email,
    program: "",
    internshipField: "",
    startDate1: "",
    endDate1: "",
    startDate2: "",
    endDate2: "",
    arabicDepartment: ".....",
    englishDepartment: "......",
    coordinatorName: "",
    coordinatorPosition: "...... | College of Information Technology | University of Bahrain",
    coordinatorTel: "+973 ",
    coordinatorEmailAddress: "",
    poBox: "P.O. Box .....",
    universityWebsite: "www.uob.edu.bh"
  });

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setEditableFields(prev => ({ ...prev, [name]: value }));
  };

  const handleSend = async () => {
    console.log("application.id", application.id);
    if (!coordinatorUsername) {
      alert('Error: Could not identify the sender. Please log in again.');
      return;
    }

    const studentEmail = application.email;
    const letterElement = document.getElementById('trainingLetterContent');

    if (!letterElement) {
      alert('Error: Could not find the letter content to generate the PDF.');
      return;
    }

    try {
      const canvas = await html2canvas(letterElement, {
        scale: 2, // Higher scale for better quality
        useCORS: true,
      });

      const pdf = new jsPDF({
        orientation: 'p',
        unit: 'mm',
        format: 'a4',
      });

      const imgData = canvas.toDataURL('image/png');
      const pdfWidth = pdf.internal.pageSize.getWidth();
      const pdfHeight = pdf.internal.pageSize.getHeight();
      const canvasWidth = canvas.width;
      const canvasHeight = canvas.height;
      const ratio = canvasWidth / canvasHeight;
      const width = pdfWidth;
      const height = width / ratio;

      pdf.addImage(imgData, 'PNG', 0, 0, width, height > pdfHeight ? pdfHeight : height);

      const docName = `training-letter-${application.studentId}-${Date.now()}.pdf`;
      const pdfBlob = pdf.output('blob');
      const letterFile = new File([pdfBlob], docName, { type: 'application/pdf' });

      const formData = new FormData();
      formData.append('file', letterFile);

      // 3. Upload the file
      const token = localStorage.getItem('token');
      const uploadResponse = await fetch('http://localhost:8080/api/documents/upload', {
        method: 'POST',
        body: formData,
        headers: { 'Authorization': `Bearer ${token}` },
      });

      if (!uploadResponse.ok) {
        const errorText = await uploadResponse.text();
        throw new Error(`Upload failed: ${errorText}`);
      }

      // 4. Submit document metadata
      const reader = new FileReader();
      reader.readAsBinaryString(pdfBlob);
      reader.onloadend = async () => {
        const fileContent = reader.result as string;
        const simpleHash = (str: string): string => {
          let hash = 0;
          for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash |= 0;
          }
          return 'h' + hash.toString();
        };
        const docHash = simpleHash(fileContent);
        const docId = docName; // Use the unique filename as the docId

        const payload = {
          id: docId,
          name: docName,
          hash: docHash,
          uploader: coordinatorUsername,
          approvers: [studentEmail],
        };

        try {
          await apiFetch('/api/documents', {
            method: 'POST',
            body: JSON.stringify(payload),
          });

          // Mark the notification as read
          await apiFetch(`/api/training-notifications/${application.id}/view`, {
            method: 'POST',
          });

          // Update the application status to "Completed"
          await apiFetch(`/api/applications/${application.id}/status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status: 'Completed' }),
          });

          // Dispatch a custom event to refresh notifications
          window.dispatchEvent(new CustomEvent('refreshNotifications'));

          alert('Letter has been sent to the student successfully.');
          onHide();
        } catch (error: any) {
          console.error('Error submitting document metadata:', error);
          alert(`Failed to send letter: ${error.message}`);
        }
      };
    } catch (error) {
      console.error('Error generating or uploading PDF:', error);
      alert(`Failed to generate or send letter: ${error}`);
    }
  };

  return (
    <Modal show={show} onHide={onHide} size="xl" centered>
      <Modal.Header closeButton>
        <Modal.Title>Edit Training Letter</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <div id="trainingLetterContent" style={{ 
            width: '210mm',
            minHeight: '297mm',
            padding: '25mm', 
            margin: '0 auto',
            border: '1px solid #eee',
            boxShadow: '0 0 5px rgba(0,0,0,0.1)',
            backgroundColor: '#f5ebdc',
            fontFamily: '"Times New Roman", Times, serif',
            fontSize: '12pt',
            lineHeight: '1.5'
          }}>

          {/* Headers */}
<div 
  style={{ 
    position: 'relative', 
    height: '100px', 
    marginBottom: '20px', 
    display: 'flex', 
    justifyContent: 'space-between',
    borderBottom: '2px solid #ebed64', // <-- RED LINE BELOW HEADER
    paddingBottom: '10px' // spacing between content and line
  }}
>
  {/* Left Logo + English text */}
  <div style={{ display: 'flex', alignItems: 'center' }}>
    <img src="/uob2030.png" alt="UOB 2030" style={{ height: '80px', marginLeft: '-75px', marginTop: '-75px' }} />
    <div style={{ textAlign: 'left', marginRight: '10px', marginBottom: '55px' }}>
      <div style={{ fontWeight: 'bold' }}>UNIVERSITY OF BAHRAIN</div>
      <div style={{ fontWeight: 'bold' }}>COLLEGE OF INFORMATION TECHNOLOGY</div>
      <input
        type="text"
        name="englishDepartment"
        value={editableFields.englishDepartment}
        onChange={handleChange}
        style={{
          border: 'none',
          textAlign: 'left',
          background: 'transparent',
          width: '100%'
        }}
      />
    </div>
  </div>

  {/* Arabic text + Right logo */}
  <div style={{ textAlign: 'right', display: 'flex', alignItems: 'center' }}>
    <div style={{ textAlign: 'right', marginLeft: '10px', marginBottom: '55px' }}>
      <div style={{ textAlign: 'right', fontFamily: "'Traditional Arabic', serif", fontSize: '16pt', fontWeight: 'bold' }}>
        جامعة البحرين
      </div>
      <div style={{ textAlign: 'right', fontFamily: "'Traditional Arabic', serif", fontSize: '16pt', fontWeight: 'bold' }}>
        كلية تقنية المعلومات
      </div>
      <input
        type="text"
        name="arabicDepartment"
        value={editableFields.arabicDepartment}
        onChange={handleChange}
        style={{
          border: 'none',
          textAlign: 'right',
          background: 'transparent',
          width: '100%'
        }}
      />
    </div>
    <img
      src="/uob_logo.png"
      alt="UOB Logo"
      style={{
        height: '80px',
        marginRight: '-75px',
        marginTop: '-75px',
      }}
    />
  </div>
</div>


          {/* Date */}
          <p style={{ textAlign: 'left', marginTop: '20px' }}>
            Date: {editableFields.date}
          </p>

          <h4 style={{ textAlign: 'center', marginTop: '20px' }}>TO WHOM IT MAY CONCERN</h4>

          <p>Dear Sir/Madam,</p>
          <p><strong>Subject:</strong> Internship / Training Position Seeking</p>

          <p>
            The student below is currently enrolled in the
            <input type="text" name="program" value={editableFields.program} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '250px', background: 'transparent' }} />
            program offered by the Department of Computer Engineering, College of Information Technology:
          </p>

          {/* Student Info Table */}
          <table style={{ width: '100%', borderCollapse: 'collapse', margin: '15px 0' }}>
            <tbody>
              {[
                { label: "Name", field: "studentName" },
                { label: "Student ID", field: "studentId" },
                { label: "Student CPR", field: "cpr" },
                { label: "Nationality", field: "nationality" },
                { label: "Telephone", field: "telephone" },
                { label: "E-mail", field: "email" }
              ].map(row => (
                <tr key={row.field}>
                  <td style={{ border: '1px solid #000', padding: '5px', fontWeight: 'bold', width: '30%' }}>{row.label}</td>
                  <td style={{ border: '1px solid #000', padding: '5px' }}>
                    <input
                      type="text"
                      name={row.field}
                      value={editableFields[row.field as keyof typeof editableFields]}
                      onChange={handleChange}
                      style={{ border: 'none', width: '100%', background: 'transparent' }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <p>
            The student is seeking a full-time internship opportunity in the field of
            <input type="text" name="internshipField" value={editableFields.internshipField} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px #000', margin: '0 5px', width: '300px', background: 'transparent' }} />
            as a partial fulfilment for the program requirements. Note that the internship is limited to the
            field of networking and Information Technology.
          </p>

          <p>
            Please note that this letter is NOT a personal recommendation for the above student to undertake an internship at your establishment, and it is valid only for an internship duration of full time of two months (8 weeks) which should start from
            <input type="text" name="startDate1" value={editableFields.startDate1} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '100px', background: 'transparent' }} />
            to
            <input type="text" name="endDate1" value={editableFields.endDate1} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '100px', background: 'transparent' }} />
            , or from
            <input type="text" name="startDate2" value={editableFields.startDate2} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '100px', background: 'transparent' }} />
            to
            <input type="text" name="endDate2" value={editableFields.endDate2} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '100px', background: 'transparent' }} />
          </p>

          <p>This letter is given to the student upon his/her request.</p>

          <p>
            For any queries, please contact the training course coordinator:
            <input type="text" name="coordinatorName" value={editableFields.coordinatorName} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '200px', background: 'transparent' }} />
            Tel:
            <input type="text" name="coordinatorTel" value={editableFields.coordinatorTel} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '150px', background: 'transparent' }} />.
            Email:
            <input type="text" name="coordinatorEmailAddress" value={editableFields.coordinatorEmailAddress} onChange={handleChange}
              style={{ border: 'none', borderBottom: '1px solid #000', margin: '0 5px', width: '250px', background: 'transparent' }} />
          </p>

          {/* Signature */}
          <div style={{ marginTop: '50px' }}>
            <p style={{ fontWeight: 'bold', margin: '0' }}>
              <input type="text" name="coordinatorName" value={editableFields.coordinatorName} onChange={handleChange}
                style={{ border: 'none', background: 'transparent', width: '100%' }} />
            </p>
            <p style={{ margin: '0' }}>
              <input type="text" name="coordinatorPosition" value={editableFields.coordinatorPosition} onChange={handleChange}
                style={{ border: 'none', background: 'transparent', width: '100%' }} />
            </p>
            <p style={{ margin: '0' }}>
              E-mail:
              <input type="text" name="coordinatorEmailAddress" value={editableFields.coordinatorEmailAddress} onChange={handleChange}
                style={{ border: 'none', background: 'transparent', width: '250px', margin: '0 5px' }} />
              |
              <input type="text" name="poBox" value={editableFields.poBox} onChange={handleChange}
                style={{ border: 'none', background: 'transparent', width: '150px', margin: '0 5px' }} />
              | Kingdom of Bahrain |
              <input type="text" name="universityWebsite" value={editableFields.universityWebsite} onChange={handleChange}
                style={{ border: 'none', background: 'transparent', width: '200px', margin: '0 5px' }} />
            </p>
          </div>

        </div>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={onHide}>Close</Button>
        <Button variant="primary" onClick={handleSend}><FontAwesomeIcon icon={faShare} /> Send</Button>
      </Modal.Footer>
    </Modal>
  );
};


  return (
    <Container>
      <h2 className="my-4 text-center text-primary">Training Application</h2>
      <Row className="justify-content-center">
        <Col md={10}>
          <Card className="shadow-sm">
            <Card.Body>
              <Card.Title className="text-primary">
                {userRole === 'student' ? 'Student Training Application' : 'Training Dashboard'}
              </Card.Title>
              <Card.Text>
                {userRole === 'student'
                  ? ''
                  : 'Review and manage student training applications.'}
              </Card.Text>
              {userRole === 'student' ? renderStudentView() : renderFacultyView()}
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </Container>
  );
};

export default Training;