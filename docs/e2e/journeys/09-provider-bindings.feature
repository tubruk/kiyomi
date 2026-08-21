@provider-bindings
Feature: Provider Bindings

  As a user, I want to bind multiple providers to a manga in my library
  so that I can switch content sources or enrich metadata without losing
  my existing data.

  Background:
    Given a seeded library with manga-x bound to mock-primary is running
    And I open the library
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    And the manga has providers "mock-primary"

  @smoke
  Scenario: Provider section is collapsed by default
    Then the providers section is collapsed
    And the providers section shows the "Add provider" button

  Scenario: Expand provider section to see bindings
    Given the manga has providers "mock-primary" and "mock-secondary"
    And "mock-primary" is the active content provider
    When I expand the providers section
    Then I see the provider "mock-primary" with manga_title "Alpha Manga"
    And I see the provider "mock-secondary" with manga_title "Alpha (Secondary)"
    And the content capability pill on "mock-primary" shows the active indicator
    And the content capability pill on "mock-secondary" does not show the active indicator
    And the provider "mock-primary" shows capability badges "Content"
    And the provider "mock-secondary" shows capability badges "Content"

  Scenario: Bind additional provider
    When I expand the providers section
    And I click "Add provider"
    And the add provider dialog opens
    And I select provider "mock-secondary" in the dialog
    And I search the provider dialog for "Alpha"
    And I click the result for "mock-secondary"
    And I confirm the dialog
    Then the provider "mock-secondary" is added to the manga
    And "mock-primary" remains the active content provider

  Scenario: Per-row switch warns then wipes cached chapters
    Given the manga has providers "mock-primary" and "mock-secondary"
    And "mock-primary" is the active content provider
    When I expand the providers section
    And I click "Switch content to this" on provider "mock-secondary"
    Then a confirmation dialog appears warning about discarding cached chapters
    When I confirm the destructive switch
    Then "mock-secondary" becomes the active content provider
    And no add provider dialog is opened
    And chapters are refreshed from "mock-secondary"

  Scenario: Switch button visibility on rows
    Given the manga has providers "mock-primary" and "mock-secondary"
    And "mock-primary" is the active content provider
    When I expand the providers section
    Then the provider "mock-primary" (active) has no "Switch content to this" action
    And the provider "mock-secondary" (non-active) has the "Switch content to this" action

  Scenario: Remove provider binding
    Given the manga has providers "mock-primary" and "mock-secondary"
    And "mock-primary" is the active content provider
    When I expand the providers section
    And I click "Remove" on provider "mock-secondary"
    Then the provider "mock-secondary" is removed from the manga
    And "mock-primary" remains the active content provider

  Scenario: Cannot remove last content provider
    Given the manga has only provider "mock-primary"
    When I expand the providers section
    Then the "Remove" action on provider "mock-primary" is disabled

  Scenario: Reject setting content provider to a metadata-only binding via API
    When I PATCH the manga's content provider with a metadata-only binding
    Then the API returns 400 with "lacks content capability"
    And the active content provider is unchanged

  Scenario: Provider list menu no longer offers refresh actions
    Given the manga has providers "mock-primary" and "mock-secondary"
    And "mock-primary" is the active content provider
    When I expand the providers section
    Then no provider row offers a "Refresh chapters" action
    And no provider row offers a "Refresh metadata" action

  Scenario: Import metadata is accessed from the metadata card menu
    When I open the metadata options menu
    Then I see the "Import metadata" menu item
    And "Import metadata" is not shown in the providers section header

  Scenario: Refresh chapters is in the chapter list toolbar
    Then the "Refresh chapters" button is visible in the chapter list toolbar
