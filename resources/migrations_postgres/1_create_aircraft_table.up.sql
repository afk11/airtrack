create table aircraft (
                            id serial not null primary key,
                            icao varchar(6) not null,
                            created_at timestamp null,
                            updated_at timestamp null
);
create unique index aircraft_icao_unique on aircraft(icao);