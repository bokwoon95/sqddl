CREATE TABLE employee (
    employee_id BINARY(16)
    ,name NVARCHAR(255) NOT NULL
    ,title NVARCHAR(255) NOT NULL
    ,manager_id BINARY(16)

    ,CONSTRAINT employee_employee_id_pkey PRIMARY KEY (employee_id)
);

CREATE INDEX employee_manager_id_idx ON employee (manager_id);

CREATE TABLE department (
    department_id BINARY(16)
    ,name NVARCHAR(255) NOT NULL

    ,CONSTRAINT department_department_id_pkey PRIMARY KEY (department_id)
);

CREATE TABLE employee_department (
    employee_id BINARY(16)
    ,department_id BINARY(16)

    ,CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id)
);

CREATE INDEX employee_department_employee_id_idx ON employee_department (employee_id);

CREATE INDEX employee_department_department_id_idx ON employee_department (department_id);

CREATE TABLE task (
    task_id BINARY(16)
    ,employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL
    ,task NVARCHAR(255) NOT NULL
    ,data NVARCHAR(MAX)
    ,data_deadline AS CAST(json_value(data, '$.deadline') AS CHAR(20))

    ,CONSTRAINT task_task_id_pkey PRIMARY KEY (task_id)
);

CREATE INDEX task_employee_id_department_id_idx ON task (employee_id, department_id);

CREATE INDEX task_task_idx ON task (task DESC) INCLUDE (employee_id, department_id);

CREATE INDEX task_data_deadline_idx ON task (data_deadline DESC) WHERE data IS NOT NULL;

ALTER TABLE employee ADD CONSTRAINT employee_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES employee (employee_id);

ALTER TABLE employee_department ADD CONSTRAINT employee_department_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES employee (employee_id);
ALTER TABLE employee_department ADD CONSTRAINT employee_department_department_id_fkey FOREIGN KEY (department_id) REFERENCES department (department_id);

ALTER TABLE task ADD CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id);
