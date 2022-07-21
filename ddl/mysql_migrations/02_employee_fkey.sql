ALTER TABLE employee
    ADD CONSTRAINT employee_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES employee (employee_id)
;

ALTER TABLE employee_department
    ADD CONSTRAINT employee_department_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES employee (employee_id)
    ,ADD CONSTRAINT employee_department_department_id_fkey FOREIGN KEY (department_id) REFERENCES department (department_id)
;

ALTER TABLE task
    ADD CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id)
;
