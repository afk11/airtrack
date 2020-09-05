create table `aircraft` (
    `id` integer not null primary key autoincrement,
    `icao` varchar(6) not null,
    `created_at` timestamp null,
    `updated_at` timestamp null
);
create unique index `aircraft_icao_unique` on aircraft(`icao`);
