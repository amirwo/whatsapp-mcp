-- Migration: Add starred column to messages table
-- Version: 001
-- Description: Add starred boolean field to support WhatsApp starred messages feature

-- Add starred column with default value false
ALTER TABLE messages ADD COLUMN starred BOOLEAN DEFAULT FALSE;

-- Create indexes for efficient starred message queries
CREATE INDEX IF NOT EXISTS idx_messages_starred ON messages(starred);
CREATE INDEX IF NOT EXISTS idx_messages_starred_chat ON messages(chat_jid, starred);
CREATE INDEX IF NOT EXISTS idx_messages_starred_timestamp ON messages(starred, timestamp DESC);

-- Update existing messages to have starred = false (for clarity, though default handles this)
UPDATE messages SET starred = FALSE WHERE starred IS NULL;