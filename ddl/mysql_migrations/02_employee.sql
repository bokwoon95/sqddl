CREATE TABLE employee (
    employee_id BINARY(16)
    ,name VARCHAR(255) NOT NULL
    ,title VARCHAR(255) NOT NULL
    ,manager_id BINARY(16)

    ,PRIMARY KEY (employee_id)
    ,INDEX employee_manager_id_idx (manager_id)
);


CREATE TABLE department (
    department_id BINARY(16)
    ,name VARCHAR(255) NOT NULL

    ,PRIMARY KEY (department_id)
);

CREATE TABLE employee_department (
    employee_id BINARY(16)
    ,department_id BINARY(16)

    ,PRIMARY KEY (employee_id, department_id)
    ,INDEX employee_department_employee_id_idx (employee_id)
    ,INDEX employee_department_department_id_idx (department_id)
);

CREATE TABLE task (
    task_id BINARY(16)
    ,employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL
    ,task VARCHAR(255) NOT NULL
    ,data JSON

    ,PRIMARY KEY (task_id)
    ,INDEX task_employee_id_department_id_idx (employee_id, department_id)
    ,INDEX task_task_idx (task DESC)
    ,INDEX task_data_idx ((CAST(json_unquote(json_extract(data, '$.deadline')) AS CHAR(20))) DESC)
);
