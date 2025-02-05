#!/usr/bin/env python
# -*- coding: utf-8 -*-
from RealtimeSTT import AudioToTextRecorder

def process_text(text):
    print(text)

if __name__ == '__main__':
    recorder = AudioToTextRecorder(wake_words="jarvis")

    while True:
        recorder.text(process_text)