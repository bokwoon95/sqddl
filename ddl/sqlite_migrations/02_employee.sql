CREATE TABLE employee (
    employee_id UUID NOT NULL
    ,name TEXT NOT NULL
    ,title TEXT NOT NULL
    ,manager_id UUID

    ,CONSTRAINT employee_employee_id_pkey PRIMARY KEY (employee_id)
    ,CONSTRAINT employee_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES employee (employee_id)
);

CREATE INDEX employee_manager_id_idx ON employee (manager_id);

CREATE TABLE department (
    department_id UUID NOT NULL
    ,name TEXT NOT NULL

    ,CONSTRAINT department_department_id_pkey PRIMARY KEY (department_id)
);

CREATE TABLE employee_department (
    employee_id UUID NOT NULL
    ,department_id UUID NOT NULL

    ,CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id)
    ,CONSTRAINT employee_department_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES employee (employee_id)
    ,CONSTRAINT employee_department_department_id_fkey FOREIGN KEY (department_id) REFERENCES department (department_id)
);

CREATE INDEX employee_department_employee_id_idx ON employee_department (employee_id);

CREATE INDEX employee_department_department_id_idx ON employee_department (department_id);

CREATE TABLE task (
    task_id UUID NOT NULL
    ,employee_id UUID NOT NULL
    ,department_id UUID NOT NULL
    ,task TEXT NOT NULL
    ,data JSON

    ,CONSTRAINT task_task_id_pkey PRIMARY KEY (task_id)
    ,CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id)
);

CREATE INDEX task_employee_id_department_id_idx ON task (employee_id, department_id);

CREATE INDEX task_task_idx ON task (task DESC);

CREATE INDEX task_data_idx ON task (json_extract(data, '$.deadline') DESC) WHERE data IS NOT NULL;
