-- Create organizations table to store domain and LDAP configuration
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    domain VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    ldap_url VARCHAR(500) NOT NULL,
    ldap_base_dn VARCHAR(500) NOT NULL,
    ldap_bind_dn VARCHAR(500),
    ldap_bind_password VARCHAR(500),
    ldap_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    CONSTRAINT organizations_domain_check CHECK (domain IS NOT NULL AND length(domain) > 0),
    CONSTRAINT organizations_name_check CHECK (name IS NOT NULL AND length(name) > 0)
);

-- Create index on domain for faster lookups
CREATE INDEX IF NOT EXISTS idx_organizations_domain ON organizations(domain);
CREATE INDEX IF NOT EXISTS idx_organizations_active ON organizations(is_active);
