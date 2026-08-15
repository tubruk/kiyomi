@explore-providers
Feature: Explore Providers and Catalogs

  As a user, I want to explore different manga providers, select a provider to view its catalog, and click a manga to view its details.

  Background:
    Given a clean server with empty library is running

  Scenario: Navigate to explore page
    Given I open the library
    When I click "Explore"
    Then I am redirected to the Explore view
    And I see the catalog header "Explore Catalog"

  Scenario: Pick provider to explore
    Given I open the Explore view
    When I select the provider "Mock Provider"
    Then the catalog for "Mock Provider" is displayed
    And I see manga titles from the provider

  Scenario: Search manga by title in provider catalog
    Given I open the Explore view
    When I select the provider "Mock Provider"
    And I search for "Alpha" in the catalog
    Then I see the manga "Alpha Manga" in search results

  @smoke
  Scenario: Click manga to see details
    Given I open the Explore view
    When I select the provider "Mock Provider"
    And I click the manga "Alpha Manga"
    Then I am on the remote manga details page for "Alpha Manga"
    And I see the manga details including synopsis and chapter list
