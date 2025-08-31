-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    fcm_token TEXT
);

-- Add role column to users table if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) NOT NULL DEFAULT 'student';

-- Add is_training_coordinator column to users table if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_training_coordinator BOOLEAN DEFAULT FALSE;
    role VARCHAR(50) NOT NULL DEFAULT 'student',
    is_training_coordinator BOOLEAN DEFAULT FALSE
);

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    doc_id TEXT UNIQUE NOT NULL,
    doc_name TEXT,
    doc_path TEXT,
    hash TEXT,
    uploader_username TEXT REFERENCES users(username),
    upload_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    approved_count INTEGER DEFAULT 0,
    rejected_count INTEGER DEFAULT 0,
    pending_count INTEGER DEFAULT 0
);

-- Create ENUM type for document_shares status
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'approver_status') THEN
        CREATE TYPE approver_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED');
    END IF;
END$$;


-- Create document_shares table
CREATE TABLE IF NOT EXISTS document_shares (
    id SERIAL PRIMARY KEY,
    document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
    receiver_username TEXT REFERENCES users(username),
    status approver_status DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(document_id, receiver_username)
);

-- Add viewed column to document_shares
ALTER TABLE document_shares ADD COLUMN IF NOT EXISTS viewed BOOLEAN DEFAULT FALSE;

-- Create sender_notifications table
CREATE TABLE IF NOT EXISTS sender_notifications (
    id SERIAL PRIMARY KEY,
    document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
    uploader_username TEXT REFERENCES users(username),
    approver_username TEXT REFERENCES users(username),
    status approver_status,
    doc_name TEXT,
    notification_type TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    viewed BOOLEAN DEFAULT FALSE
);
