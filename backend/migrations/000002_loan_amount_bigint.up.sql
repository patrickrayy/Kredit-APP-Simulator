ALTER TABLE loan_applications
    ALTER COLUMN amount TYPE BIGINT USING amount::BIGINT;

ALTER TABLE loan_applications
    ADD CONSTRAINT loan_applications_amount_positive CHECK (amount > 0);