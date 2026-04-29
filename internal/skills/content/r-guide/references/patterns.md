# R Patterns Reference

## dplyr Pipelines

### Multi-step Transformation

```r
monthly_summary <- sales_data |>
  dplyr::filter(order_date >= as.Date("2024-01-01"), status != "cancelled") |>
  dplyr::mutate(
    month = lubridate::floor_date(order_date, "month"),
    revenue = quantity * unit_price * (1 - discount)
  ) |>
  dplyr::summarise(
    total_revenue = sum(revenue, na.rm = TRUE),
    order_count = dplyr::n(),
    unique_customers = dplyr::n_distinct(customer_id),
    .by = c(month, region)
  ) |>
  dplyr::arrange(month, dplyr::desc(total_revenue))
```

### Window Functions and Ranking

```r
ranked_products <- product_metrics |>
  dplyr::mutate(
    revenue_rank = dplyr::dense_rank(dplyr::desc(revenue)),
    pct_of_total = revenue / sum(revenue, na.rm = TRUE),
    cumulative_pct = cumsum(revenue) / sum(revenue, na.rm = TRUE),
    .by = category
  ) |>
  dplyr::filter(cumulative_pct <= 0.80)
```

### Reshaping and Cleaning

```r
# Pivot wide to long
long_data <- wide_data |>
  tidyr::pivot_longer(
    cols = dplyr::starts_with("q"),
    names_to = "quarter", names_prefix = "q",
    names_transform = as.integer, values_to = "revenue"
  )

# Clean across column types
cleaned_df <- raw_df |>
  dplyr::mutate(
    dplyr::across(where(is.character), \(x) dplyr::na_if(stringr::str_trim(x), "")),
    dplyr::across(where(is.numeric), \(x) dplyr::if_else(x < 0, NA_real_, x))
  )
```

## ggplot2 Visualization

### Faceted Time Series

```r
plot_time_series <- function(df, date_col, value_col, facet_col) {
  ggplot2::ggplot(df, ggplot2::aes(x = {{ date_col }}, y = {{ value_col }})) +
    ggplot2::geom_line(color = "#2563eb", linewidth = 0.8) +
    ggplot2::geom_smooth(method = "loess", se = TRUE, alpha = 0.2) +
    ggplot2::facet_wrap(ggplot2::vars({{ facet_col }}), scales = "free_y") +
    ggplot2::scale_x_date(date_labels = "%b %Y") +
    ggplot2::theme_minimal(base_size = 12)
}
```

### Horizontal Bar Chart with Reordering

```r
plot_bar <- function(df, x_col, y_col, fill_col = NULL) {
  ggplot2::ggplot(df, ggplot2::aes(
    x = forcats::fct_reorder({{ x_col }}, {{ y_col }}),
    y = {{ y_col }}, fill = {{ fill_col }}
  )) +
    ggplot2::geom_col(width = 0.7) +
    ggplot2::scale_y_continuous(labels = scales::comma) +
    ggplot2::theme_minimal(base_size = 12) +
    ggplot2::coord_flip()
}
```

## purrr Functional Programming

### Reading Multiple Files

```r
read_all_csvs <- function(directory) {
  fs::dir_ls(directory, glob = "*.csv") |>
    purrr::map_dfr(\(path) {
      readr::read_csv(path, show_col_types = FALSE) |>
        dplyr::mutate(source_file = fs::path_file(path), .before = 1)
    })
}
```

### Safe Execution with Error Collection

```r
safe_fetch <- purrr::safely(httr2::req_perform)
results <- urls |>
  purrr::map(\(url) httr2::request(url) |> safe_fetch()) |>
  purrr::set_names(urls)

successes <- results |> purrr::keep(\(x) is.null(x$error)) |> purrr::map("result")
failures <- results |> purrr::discard(\(x) is.null(x$error)) |> purrr::map("error")
```

### Nested Model Fitting (nest + map + broom)

```r
model_by_group <- function(df, group_col, formula) {
  df |>
    tidyr::nest(.by = {{ group_col }}) |>
    dplyr::mutate(
      model = purrr::map(data, \(d) lm(formula, data = d)),
      tidy = purrr::map(model, broom::tidy)
    ) |>
    tidyr::unnest(tidy) |>
    dplyr::select(-data, -model)
}
```
