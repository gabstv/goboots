CREATE TABLE IF NOT EXISTS `goboots_sessid` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `sid` char(36) CHARACTER SET ascii NOT NULL DEFAULT '',
  `time` datetime NOT NULL,
  `updated` datetime NOT NULL,
  `expires` datetime NOT NULL,
  `data` blob,
  `shortexpires` datetime NOT NULL,
  `shortcount` tinyint(2) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `sid` (`sid`),
  KEY `updated` (`updated`),
  KEY `expires` (`expires`),
  KEY `shortcount` (`shortcount`,`shortexpires`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;