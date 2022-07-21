ALTER TABLE employee
    DROP CONSTRAINT employee_manager_id_fkey
;

ALTER TABLE employee_department
    DROP CONSTRAINT employee_department_employee_id_fkey
    ,DROP CONSTRAINT employee_department_department_id_fkey
;

ALTER TABLE task
    DROP CONSTRAINT task_employee_id_department_id_fkey
;
