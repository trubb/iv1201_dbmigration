CREATE TABLE Role (
    role_id SERIAL PRIMARY KEY, name VARCHAR(255)
);

CREATE TABLE Person (
    person_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    surname VARCHAR(255) NOT NULL,
    ssn VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role INTEGER REFERENCES Role NOT NULL,
    username VARCHAR(255) NOT NULL,
    UNIQUE (username),
    UNIQUE (ssn),
    UNIQUE (email)
);

CREATE TABLE Availability (
    availability_id SERIAL PRIMARY KEY,
    person INTEGER REFERENCES person,
    from_date CHAR(10),
    to_date CHAR(10),
    UNIQUE (person, from_date, to_date)
);

CREATE TABLE Competence (
    competence_id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);

CREATE TABLE Competence_profile (
    competence_profile_id SERIAL PRIMARY KEY,
    person INTEGER REFERENCES person,
    competence INTEGER REFERENCES competence,
    years_of_experience INTEGER,
    UNIQUE (person, competence, years_of_experience)
);


CREATE TABLE Status (
    name VARCHAR(255) PRIMARY KEY
);

CREATE TABLE Application (
    version INTEGER,
    person INTEGER REFERENCES person PRIMARY KEY,
    status VARCHAR(255) REFERENCES status
);

INSERT INTO Status (name) VALUES ('unhandled');
INSERT INTO Status (name) VALUES ('accepted');
INSERT INTO Status (name) VALUES ('rejected');

INSERT INTO Role (name) VALUES ('recruiter');
INSERT INTO Role (name) VALUES ('applicant');

INSERT INTO Person (name, surname, ssn, email, password, role, username) VALUES ('applicant1', 'applicant1', '12345678-4321', 'applicant1@applicant.se', 'applicant1', 2, 'applicant1');
INSERT INTO Person (name, surname, ssn, email, password, role, username) VALUES ('applicant2', 'applicant2', '12345678-4322', 'applicant2@applicant.se', 'applicant2', 2, 'applicant2');
INSERT INTO Person (name, surname, ssn, email, password, role, username) VALUES ('recruiter1', 'recruiter1', '12345678-1111', 'recruiter1@recruiter.se', 'recruiter1', 1, 'recruiter1');
INSERT INTO Person (name, surname, ssn, email, password, role, username) VALUES ('recruiter2', 'recruiter2', '12345678-2222', 'recruiter2@recruiter.se', 'recruiter2', 1, 'recruiter2');

INSERT INTO Competence (name) VALUES ('kassör');
INSERT INTO Competence (name) VALUES ('kock');
INSERT INTO Competence (name) VALUES ('säljare');
INSERT INTO Competence (name) VALUES ('borgmästare');

