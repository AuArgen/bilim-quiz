ALTER TABLE teachers
  ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'teacher';

ALTER TABLE teachers
  DROP CONSTRAINT IF EXISTS teachers_role_check;

ALTER TABLE teachers
  ADD CONSTRAINT teachers_role_check CHECK (role IN ('teacher', 'admin'));

CREATE INDEX IF NOT EXISTS idx_teachers_role ON teachers(role);
