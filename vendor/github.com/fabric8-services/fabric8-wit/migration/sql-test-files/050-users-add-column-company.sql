-- Set company value to the existing users
UPDATE users SET company='RedHat Inc.' WHERE full_name='test one' OR full_name='test two';
