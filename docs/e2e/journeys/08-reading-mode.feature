@reading-mode
Feature: Reading Mode Configuration and Reader Layout

  As a user, I want manga reading modes to be extracted from providers and configurable per manga,
  so that the reader renders chapters in the appropriate layout (RTL, LTR, Vertical, or Longstrip)
  and falls back to defaults when unspecified.

  Background:
    Given a seeded library with manga-x is running
    And I open the library

  Scenario: Provider metadata extraction and default reading mode normalization
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I toggle detailed metadata
    Then I see the manga reading mode is "Right to Left (Manga)"

  @smoke
  Scenario: Edit reading mode to Longstrip via Edit Metadata dialog and persist across navigation
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Longstrip (Webtoon)"
    And I save the metadata changes
    When I toggle detailed metadata
    Then I see the manga reading mode is "Longstrip (Webtoon)"
    When I click "Back to Library"
    Then I am on the library page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I toggle detailed metadata
    Then I see the manga reading mode is "Longstrip (Webtoon)"

  Scenario: Edit reading mode to Vertical Gapped via Edit Metadata dialog
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Vertical (Gapped)"
    And I save the metadata changes
    When I toggle detailed metadata
    Then I see the manga reading mode is "Vertical (Gapped)"

  Scenario: Edit reading mode to Left to Right via Edit Metadata dialog
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Left to Right (Comic)"
    And I save the metadata changes
    When I toggle detailed metadata
    Then I see the manga reading mode is "Left to Right (Comic)"

  Scenario: Reader layout for Longstrip continuous scroll
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Longstrip (Webtoon)"
    And I save the metadata changes
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    And the reader layout is rendered as continuous longstrip scroll

  Scenario: Reader layout for Vertical gapped scroll
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Vertical (Gapped)"
    And I save the metadata changes
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    And the reader layout is rendered as vertical gapped scroll

  Scenario: Reader layout for Right to Left paginated reading
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Right to Left (Manga)"
    And I save the metadata changes
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    And the reader layout is rendered as right-to-left

  Scenario: Reader layout for Left to Right paginated reading
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I open the edit metadata dialog
    And I select the reading mode option "Left to Right (Comic)"
    And I save the metadata changes
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    And the reader layout is rendered as left-to-right

  Scenario: Fallback to default reading mode when unspecified
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    And the reader falls back to the default reading mode layout

  Scenario: Touch swipe navigation across paginated reader
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I click "Start Reading"
    Then I am in the reader for "Alpha Manga"
    And I am on page 1 of chapter 1
    When I swipe right on the reader canvas
    Then I am on page 2 of chapter 1
    When I swipe left on the reader canvas
    Then I am on page 1 of chapter 1

  Scenario: Swiping backward to previous chapter lands on the last page
    When I click on the manga "Alpha Manga" in the library
    Then I am on the library manga details page for "Alpha Manga"
    When I click on chapter "Chapter 2"
    Then I am in the reader for "Alpha Manga"
    And I am on page 1 of chapter 2
    When I swipe left on the reader canvas
    Then I am on the last page of chapter 1
