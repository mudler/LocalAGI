from configparser import ConfigParser

config_file = "config.ini"
config = ConfigParser(interpolation=None)
config.read(config_file)