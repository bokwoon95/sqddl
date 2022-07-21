ALTER TABLE person VALIDATE CONSTRAINT person_email_not_null_check;
ALTER TABLE person ALTER COLUMN email SET NOT NULL;
ALTER TABLE person DROP CONSTRAINT person_email_not_null_check;
