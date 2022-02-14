CREATE TABLE records (
  id serial not null unique,
  title varchar(255) not null,
  artist varchar(255) not null,
  price integer not null
);