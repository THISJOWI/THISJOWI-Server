-- Add OrgID and LDAP columns to users table
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS org_id UUID,
ADD COLUMN IF NOT EXISTS ldap_username VARCHAR(255),
ADD COLUMN IF NOT EXISTS is_ldap_user BOOLEAN NOT NULL DEFAULT FALSE,
ADD CONSTRAINT fk_users_org_id FOREIGN KEY (org_id) REFERENCES organizations(id);

-- Create unique index on email per organization (allows same email in different orgs)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_org_id ON users(email, org_id) WHERE org_id IS NOT NULL;

-- For regular users without org
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_no_org ON users(email) WHERE org_id IS NULL;

-- Index for LDAP lookups
CREATE INDEX IF NOT EXISTS idx_users_ldap_username ON users(ldap_username);
CREATE INDEX IF NOT EXISTS idx_users_ldap_user ON users(is_ldap_user);
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(org_id);
