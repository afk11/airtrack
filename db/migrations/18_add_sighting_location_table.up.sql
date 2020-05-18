create table `sighting_location` (`id` int unsigned not null auto_increment primary key, `sighting_id` int not null, `timestamp` timestamp not null, `altitude` mediumint not null, `latitude` double(12, 8) not null, `longitude` double(12, 8) not null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';