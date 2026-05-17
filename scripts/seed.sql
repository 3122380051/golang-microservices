INSERT INTO roles (name, description)
VALUES
    ('admin', 'System administrator'),
    ('user', 'Standard trading user'),
    ('trader', 'Active trader role')
ON CONFLICT (name) DO NOTHING;
