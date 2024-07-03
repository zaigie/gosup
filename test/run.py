import signal
import time
from datetime import datetime

start_time = time.time()


def signal_handler(sig, _frame):
    end_time = time.time()
    elapsed_time = end_time - start_time
    end_datetime = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    with open("stop.log", "a", encoding="utf-8") as f:
        f.write(f"\nElapsed Time: {elapsed_time:.2f}s")
        f.write(f"\nEnd Time: {end_datetime}")
        f.write(f"\nAbort Signal: {sig}")
    exit(0)


for _sig in [signal.SIGINT, signal.SIGTERM, signal.SIGHUP, signal.SIGQUIT]:
    signal.signal(_sig, signal_handler)

if __name__ == "__main__":
    print("Subprocess Running...")
    print(f"Start Time: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

    while True:
        time.sleep(1)
        print(f"{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
