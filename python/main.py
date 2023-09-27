from fastapi import FastAPI
from apscheduler.schedulers.asyncio import AsyncIOScheduler
import httpx
from typing import List
from collections import defaultdict
from telegram import Bot
import os

TELEGRAM_TOKEN = os.getenv("TELEGRAM_TOKEN")
CHAT_ID = os.getenv("CHAT_ID")

app = FastAPI()

# Initialize the Telegram Bot
bot = Bot(TELEGRAM_TOKEN)

# VPS Servers and the services running on them
vps_servers = {
    "vps_1": [
        "https://project.web-ar.studio/health",
        "http://",
    ],
    # "vps_2": ["http://example2.com", "http://service2.example.com"],
    # Add more VPS and services as needed
}

# Store failed request counts
failed_counts = defaultdict(int)


async def check_service_status(url: str):
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(url)
            if response.status_code == 200:
                failed_counts[url] = 0  # Reset failure count on success
                return {"url": url, "status": "UP"}
            else:
                failed_counts[url] += 1
                return {"url": url, "status": "DOWN", "code": response.status_code}
        except httpx.RequestError:
            failed_counts[url] += 1
            return {"url": url, "status": "DOWN", "error": "Unable to connect"}


async def periodic_check():
    results = {}
    for vps_name, services in vps_servers.items():
        results[vps_name] = await check_services(services)
    print(results)  # Replace this with proper logging or other actions


async def check_services(services: List[str]):
    results = []
    for url in services:
        result = await check_service_status(url)
        if failed_counts[url] >= 3:
            # Send message to Telegram
            await bot.send_message(chat_id=CHAT_ID, text=f"Service {url} is down!")
            failed_counts[url] = 0  # Reset failure count after sending message
        results.append(result)
    return results


@app.get("/")
def read_root():
    return {"message": "Service status checker"}


@app.get("/check/")
async def check_status(vps: List[str] = None):
    result = {}

    if vps is None:
        # Check all VPS
        for vps_name, services in vps_servers.items():
            result[vps_name] = await check_services(services)
    else:
        # Check selected VPS
        for vps_name in vps:
            if vps_name in vps_servers:
                result[vps_name] = await check_services(vps_servers[vps_name])

    return result


# Schedule the periodic check
scheduler = AsyncIOScheduler()
scheduler.add_job(periodic_check, "interval", minutes=1)
scheduler.start()
