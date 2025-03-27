import json
import logging
import logging.handlers
import time
import urllib.parse
from typing import Dict, List, Set, Any

import requests


def load_config(config_path: str) -> Dict[str, Any]:
    """Load configuration from JSON file.

    Args:
        config_path: Path to the configuration file

    Returns:
        Dictionary containing configuration values
    """
    try:
        with open(config_path, "r", encoding="utf-8") as config_file:
            return json.load(config_file)
    except FileNotFoundError as e:
        raise FileNotFoundError(f"Config file not found: {config_path}") from e
    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON in config file: {config_path}") from e


def setup_logger(log_filename: str) -> logging.Logger:
    """Set up and configure logger.

    Args:
        log_filename: Name of the log file

    Returns:
        Configured logger instance
    """
    logger = logging.getLogger("FeedlyAssetIntegrator")
    logger.setLevel(logging.DEBUG)

    handler = logging.handlers.RotatingFileHandler(
        log_filename, maxBytes=5 * 1024 * 1024, backupCount=5
    )
    formatter = logging.Formatter("%(asctime)s - %(levelname)s - %(message)s")
    handler.setFormatter(formatter)
    logger.addHandler(handler)

    return logger


def fetch_jira_data(
    jira_url: str,
    aql_query: str,
    page_size: int,
    headers: Dict[str, str],
    logger: logging.Logger,
) -> List[Dict[str, Any]]:
    """Fetch data from Jira API using AQL query.

    Args:
        jira_url: Base URL for Jira API
        aql_query: AQL query string
        page_size: Number of results per page
        headers: HTTP headers for the request
        logger: Logger instance

    Returns:
        List of object entries from Jira
    """
    all_entries = []
    current_page = 1
    encoded_aql_query = urllib.parse.quote(aql_query)

    while True:
        url = f"{jira_url}/rest/assets/latest/aql/objects?resultPerPage={page_size}&page={current_page}&qlQuery={encoded_aql_query}"
        logger.debug("Fetching data from Jira API: %s", url)

        try:
            response = requests.get(url, headers=headers, timeout=30)
            response.raise_for_status()
        except requests.RequestException as e:
            logger.error("Error fetching data: %s", str(e))
            break

        data = response.json()
        object_entries = data.get("objectEntries", [])
        all_entries.extend(object_entries)

        page_number = data.get("pageNumber", 1)
        total_pages = data.get("pageSize", 1)

        if page_number >= total_pages:
            break

        current_page += 1

    logger.info("Total entries fetched: %d", len(all_entries))
    return all_entries


def process_entries(
    entries: List[Dict[str, Any]], logger: logging.Logger
) -> Dict[str, List[str]]:
    """Process Jira entries into object type lists.

    Args:
        entries: List of entries from Jira
        logger: Logger instance

    Returns:
        Dictionary mapping object types to lists of labels
    """
    object_type_lists = {}

    if not entries:
        logger.warning("No entries found to process.")
        return object_type_lists

    logger.debug("Starting to process %d entries.", len(entries))

    for entry in entries:
        object_type_name = entry.get("objectType", {}).get("name")
        if not object_type_name:
            logger.warning("Entry does not have an objectType name.")
            continue

        label_value = entry.get("label")

        if label_value:
            object_type_lists.setdefault(object_type_name, []).append(label_value)
        else:
            logger.warning("No 'label' attribute found for entry: %s", entry)

    logger.debug("Processed entries into object type lists: %s", object_type_lists)
    return object_type_lists


def fetch_feedly_data(
    upload_url: str, headers: Dict[str, str], logger: logging.Logger
) -> List[Dict[str, Any]]:
    """Fetch data from Feedly API.

    Args:
        upload_url: URL for Feedly API
        headers: HTTP headers for the request
        logger: Logger instance

    Returns:
        List of items from Feedly
    """
    try:
        response = requests.get(
            f"{upload_url}?details=true", headers=headers, timeout=30
        )
        response.raise_for_status()
        logger.debug("Successfully retrieved data from feedly")
        return response.json()
    except requests.RequestException as e:
        logger.error("Error fetching data from Feedly: %s", str(e))
        return []


def sync_to_feedly(
    jira_data: Dict[str, List[str]],
    feedly_data: List[Dict[str, Any]],
    upload_url: str,
    headers: Dict[str, str],
    logger: logging.Logger,
    test_mode: bool = False,
) -> None:
    """Synchronize Jira data to Feedly.

    Args:
        jira_data: Dictionary mapping object types to lists of labels
        feedly_data: List of items from Feedly
        upload_url: URL for Feedly API
        headers: HTTP headers for the request
        logger: Logger instance
        test_mode: Whether to run in test mode (no actual API calls)
    """
    try:
        feedly_entities = {
            item["label"]: {entity.get("text") for entity in item.get("entities", [])}
            for item in feedly_data
        }
        logger.debug("Feedly entities structure: %s", feedly_entities)

        list_counts = {
            object_type: len(
                [label for label in feedly_entities if label.startswith(object_type)]
            )
            for object_type in jira_data
        }

        for object_type, names in jira_data.items():
            logger.debug(
                "Processing object type: %s with names: %s", object_type, names
            )

            existing_lists = [
                item for item in feedly_data if item["label"].startswith(object_type)
            ]
            new_entries = set(names)

            for item in existing_lists:
                current_entities = {
                    entity.get("text") for entity in item.get("entities", [])
                }
                new_entries -= current_entities

            _add_entries_to_feedly(
                new_entries,
                existing_lists,
                object_type,
                list_counts,
                upload_url,
                headers,
                logger,
                test_mode,
            )

    except (KeyError, TypeError) as e:
        logger.error("Data structure error in sync_to_feedly: %s", str(e))
    except ValueError as e:
        logger.error("Value error in sync_to_feedly: %s", str(e))


def _add_entries_to_feedly(
    new_entries: Set[str],
    existing_lists: List[Dict[str, Any]],
    object_type: str,
    list_counts: Dict[str, int],
    upload_url: str,
    headers: Dict[str, str],
    logger: logging.Logger,
    test_mode: bool,
) -> None:
    """Add new entries to Feedly lists.

    Args:
        new_entries: Set of new entries to add
        existing_lists: List of existing Feedly lists
        object_type: Type of object being processed
        list_counts: Counts of existing lists by object type
        upload_url: URL for Feedly API
        headers: HTTP headers for the request
        logger: Logger instance
        test_mode: Whether to run in test mode (no actual API calls)
    """
    while new_entries:
        added = False
        for item in existing_lists:
            if len(item["entities"]) < 50:
                current_entities = {
                    entity.get("text") for entity in item.get("entities", [])
                }

                to_add = list(new_entries - current_entities)
                if to_add:
                    space_left = 50 - len(item["entities"])
                    to_add = to_add[:space_left]

                    item["entities"].extend(
                        {"type": "customKeyword", "text": name} for name in to_add
                    )
                    payload = {
                        "id": item["id"],
                        "label": item["label"],
                        "entities": item["entities"],
                        "type": "customTopic",
                    }

                    if test_mode:
                        logger.info(
                            "Test mode: Prepared PUT request payload: %s",
                            json.dumps(payload, indent=2),
                        )
                    else:
                        _update_feedly_list(
                            upload_url, payload, headers, item["label"], logger
                        )
                        time.sleep(1)

                    new_entries -= set(to_add)
                    added = True
                    break

        if not added and new_entries:
            _create_new_feedly_list(
                new_entries,
                object_type,
                list_counts,
                upload_url,
                headers,
                logger,
                test_mode,
            )


def _update_feedly_list(
    upload_url: str,
    payload: Dict[str, Any],
    headers: Dict[str, str],
    label: str,
    logger: logging.Logger,
) -> None:
    """Update an existing Feedly list.

    Args:
        upload_url: URL for Feedly API
        payload: Request payload
        headers: HTTP headers for the request
        label: Label of the list being updated
        logger: Logger instance
    """
    try:
        response = requests.put(upload_url, json=payload, headers=headers, timeout=30)
        if response.status_code == 204:
            logger.info("Added new entities to '%s'", label)
        else:
            logger.error(
                "Failed to add entities to '%s': %d - %s",
                label,
                response.status_code,
                response.text,
            )
    except requests.RequestException as e:
        logger.error("Request error updating list '%s': %s", label, str(e))


def _create_new_feedly_list(
    new_entries: Set[str],
    object_type: str,
    list_counts: Dict[str, int],
    upload_url: str,
    headers: Dict[str, str],
    logger: logging.Logger,
    test_mode: bool,
) -> None:
    """Create a new Feedly list.

    Args:
        new_entries: Set of entries to add to the new list
        object_type: Type of object being processed
        list_counts: Counts of existing lists by object type
        upload_url: URL for Feedly API
        headers: HTTP headers for the request
        logger: Logger instance
        test_mode: Whether to run in test mode (no actual API calls)
    """
    list_counts[object_type] += 1
    new_label = f"{object_type}-{list_counts[object_type]}"
    logger.debug("Creating new list '%s' for remaining entries", new_label)

    to_add = list(new_entries)[:50]
    payload = {
        "label": new_label,
        "entities": [{"type": "customKeyword", "text": name} for name in to_add],
        "type": "customTopic",
    }

    if test_mode:
        logger.info(
            "Test mode: Prepared POST request payload: %s",
            json.dumps(payload, indent=2),
        )
    else:
        try:
            response = requests.post(
                upload_url, json=payload, headers=headers, timeout=30
            )
            if response.status_code == 204:
                logger.info("Created new list '%s' and added entities", new_label)
            else:
                logger.error(
                    "Failed to create list '%s' and add entities: %d - %s",
                    new_label,
                    response.status_code,
                    response.text,
                )
        except requests.RequestException as e:
            logger.error("Request error creating list '%s': %s", new_label, str(e))
        time.sleep(1)

    new_entries -= set(to_add)


def main():
    """Main function to run the Feedly Asset Integrator."""
    try:
        config = load_config("config.json")

        jira_url = config["jira_url"]
        aql_query = config["aql_query"]
        page_size = config["page_size"]
        upload_url = config["upload_url"]
        jira_api_token = config["jira_api_token"]
        api_key = config["api_key"]

        log_filename = "feedly_asset_sync.log"
        logger = setup_logger(log_filename)

        jira_headers = {
            "Authorization": f"Bearer {jira_api_token}",
            "Content-Type": "application/json",
        }

        feedly_headers = {
            "accept": "application/json",
            "content-type": "application/json",
            "Authorization": f"Bearer {api_key}",
        }

        jira_entries = fetch_jira_data(
            jira_url, aql_query, page_size, jira_headers, logger
        )
        jira_data = process_entries(jira_entries, logger)
        feedly_data = fetch_feedly_data(upload_url, feedly_headers, logger)

        sync_to_feedly(
            jira_data, feedly_data, upload_url, feedly_headers, logger, test_mode=False
        )
    except FileNotFoundError as e:
        logger = logging.getLogger("FeedlyAssetIntegrator")
        logger.error("File not found: %s", str(e))
    except ValueError as e:
        logger = logging.getLogger("FeedlyAssetIntegrator")
        logger.error("Configuration error: %s", str(e))
    except KeyError as e:
        logger = logging.getLogger("FeedlyAssetIntegrator")
        logger.error("Missing required configuration key: %s", str(e))


if __name__ == "__main__":
    main()
