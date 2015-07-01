CREATE TABLE `network` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `address` int unsigned NOT NULL,
  `mask` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `address` (`address`, `mask`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `ip` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `network_id` bigint(20) unsigned NOT NULL,
  `address` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`network_id`) REFERENCES `network`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,
  UNIQUE KEY `address` (`address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `gateway` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `mac` char(17) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `mac` (`mac`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `host` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `ip_id` bigint(20) unsigned DEFAULT NULL,
  `mac` char(17) NOT NULL,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,
  UNIQUE KEY `ip-mac` (`ip_id`, `mac`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `router` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `ip_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `acl` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `network` int unsigned NOT NULL,
  `mask` int unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `acl` (`network`, `mask`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `vip` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `ip_id` bigint(20) unsigned NOT NULL,
  `host_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`ip_id`) REFERENCES `ip`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,
  FOREIGN KEY (`host_id`) REFERENCES `host`(`id`) ON UPDATE CASCADE ON DELETE RESTRICT,
  UNIQUE KEY `vip` (`ip_id`, `host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
