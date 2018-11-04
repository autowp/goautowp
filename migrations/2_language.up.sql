CREATE TABLE `language` (
  `code` varchar(5) NOT NULL,
  PRIMARY KEY (`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

INSERT INTO language (code) 
VALUES ("br"), ("de"), ("en"), ("es"), ("fr"), ("it"), ("pt"), ("pt-br"), ("ru"), ("uk"), ("xx"), ("zh");
