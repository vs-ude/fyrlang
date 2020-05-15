avrdude -c arduino -p atmega328p -P /dev/ttyACM0 -b 19600 -U flash:w:arduino.hex:i
