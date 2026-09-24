ALTER TABLE loan_applications
    DROP CONSTRAINT loan_applications_amount_positive;

ALTER TABLE loan_applications
    ALTER COLUMN amount TYPE NUMERIC(15,2);