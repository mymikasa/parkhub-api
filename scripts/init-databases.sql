CREATE DATABASE IF NOT EXISTS `parkhub_identity` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `parkhub_parking` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `parkhub_iot` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `parkhub_event` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `parkhub_billing` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `parkhub_payment` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `parkhub_sms` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

GRANT ALL PRIVILEGES ON `parkhub_identity`.* TO 'parkhub'@'%';
GRANT ALL PRIVILEGES ON `parkhub_parking`.* TO 'parkhub'@'%';
GRANT ALL PRIVILEGES ON `parkhub_iot`.* TO 'parkhub'@'%';
GRANT ALL PRIVILEGES ON `parkhub_event`.* TO 'parkhub'@'%';
GRANT ALL PRIVILEGES ON `parkhub_billing`.* TO 'parkhub'@'%';
GRANT ALL PRIVILEGES ON `parkhub_payment`.* TO 'parkhub'@'%';
GRANT ALL PRIVILEGES ON `parkhub_sms`.* TO 'parkhub'@'%';

FLUSH PRIVILEGES;
