create table users
(
    id serial primary key,
    created_at timestamp not null default now(),
    login varchar(255) not null unique,
    password_hash varchar(255) not null
);

create table statuses
(
    id serial primary key,
    name varchar(255) not null unique
);

insert into statuses (id, name) values
(1, 'NEW'),
(2, 'PROCESSING'),
(3, 'INVALID'),
(4, 'PROCESSED');

create table orders
(
    id serial primary key,
    uploaded_at timestamp not null default now(),
    user_id integer not null references users(id) on delete cascade,
    status_id integer not null references statuses(id),
    number varchar(255) not null,
    accrual numeric(10,2) default 0.00,
    unique(number)
);

create table withdrawals
(
    id serial primary key,
    user_id integer not null references users(id) on delete cascade,
    number varchar(255) not null,
    amount numeric(10,2) not null,
    processed_at timestamp not null default now()
);