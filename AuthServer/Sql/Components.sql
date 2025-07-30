BEGIN;

-- CREATE TABLE "Components" -----------------------------------
CREATE TABLE "public"."Components" (
                                       "ID" Integer DEFAULT nextval('"Components_ID_seq"'::regclass) NOT NULL,
                                       "RegTime" Timestamp With Time Zone NOT NULL,
                                       "ComponentID" Character Varying( 255 ),
                                       "PublisherID" Character Varying,
                                       "SnowID" BigInt,
                                       "Version" Character Varying( 255 ),
                                       "Intro" Text,
                                       "Author" Character Varying( 255 ),
                                       "StartTime" Timestamp With Time Zone,
                                       "ServerNodeID" Integer,
                                       "Pid" BigInt,
                                       "LocalIP" Character Varying( 255 ),
                                       "AccessKey" Character Varying( 255 ),
                                       CONSTRAINT "unique_Components_RegTime" UNIQUE( "RegTime" ) );
;
-- -------------------------------------------------------------

-- CHANGE "COMMENT" OF "TABLE "Components" ---------------------
COMMENT ON TABLE "public"."Components" IS '组件注册表';
-- -------------------------------------------------------------

COMMIT;
