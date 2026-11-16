-- Rollback: Remove collaborations and invitations system
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS account_collaborators;
