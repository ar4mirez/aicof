# WordPress Patterns Reference

## Contents

- [Custom Post Types](#custom-post-types)
- [Custom Taxonomies](#custom-taxonomies)
- [Meta Boxes and Custom Fields](#meta-boxes-and-custom-fields)
- [Database Operations](#database-operations)
- [Gutenberg Block (Full Example)](#gutenberg-block-full-example)
- [WooCommerce Integration](#woocommerce-integration)
- [Testing](#testing)
- [Performance and Caching](#performance-and-caching)
- [AJAX Handling](#ajax-handling)
- [Custom WP-CLI Commands](#custom-wp-cli-commands)
- [Theme Patterns](#theme-patterns)

## Custom Post Types

```php
<?php
declare(strict_types=1);

namespace MyPlugin\PostTypes;

final class BookPostType
{
    public const POST_TYPE = 'book';

    public function __construct()
    {
        add_action('init', [$this, 'register']);
    }

    public function register(): void
    {
        $labels = [
            'name'               => __('Books', 'my-plugin'),
            'singular_name'      => __('Book', 'my-plugin'),
            'menu_name'          => __('Books', 'my-plugin'),
            'add_new'            => __('Add New', 'my-plugin'),
            'add_new_item'       => __('Add New Book', 'my-plugin'),
            'edit_item'          => __('Edit Book', 'my-plugin'),
            'new_item'           => __('New Book', 'my-plugin'),
            'view_item'          => __('View Book', 'my-plugin'),
            'search_items'       => __('Search Books', 'my-plugin'),
            'not_found'          => __('No books found', 'my-plugin'),
            'not_found_in_trash' => __('No books found in trash', 'my-plugin'),
        ];

        $args = [
            'labels'              => $labels,
            'public'              => true,
            'publicly_queryable'  => true,
            'show_ui'             => true,
            'show_in_menu'        => true,
            'show_in_rest'        => true,  // Required for Gutenberg and REST API
            'query_var'           => true,
            'rewrite'             => ['slug' => 'books'],
            'capability_type'     => 'post',
            'has_archive'         => true,
            'hierarchical'        => false,
            'menu_position'       => 20,
            'menu_icon'           => 'dashicons-book',
            'supports'            => [
                'title', 'editor', 'author', 'thumbnail',
                'excerpt', 'comments', 'custom-fields',
            ],
            'template'            => [
                ['core/paragraph', ['placeholder' => 'Add book description...']],
            ],
        ];

        register_post_type(self::POST_TYPE, $args);
    }
}
```

## Custom Taxonomies

```php
<?php
declare(strict_types=1);

namespace MyPlugin\Taxonomies;

final class GenreTaxonomy
{
    public function __construct()
    {
        add_action('init', [$this, 'register']);
    }

    public function register(): void
    {
        // Hierarchical taxonomy (like categories)
        register_taxonomy('genre', 'book', [
            'labels' => [
                'name'          => __('Genres', 'my-plugin'),
                'singular_name' => __('Genre', 'my-plugin'),
            ],
            'public'            => true,
            'hierarchical'      => true,
            'show_in_rest'      => true,
            'show_admin_column' => true,
            'rewrite'           => ['slug' => 'genre'],
        ]);

        // Non-hierarchical taxonomy (like tags)
        register_taxonomy('book_author', 'book', [
            'labels' => [
                'name'          => __('Authors', 'my-plugin'),
                'singular_name' => __('Author', 'my-plugin'),
            ],
            'public'            => true,
            'hierarchical'      => false,
            'show_in_rest'      => true,
            'show_admin_column' => true,
            'rewrite'           => ['slug' => 'book-author'],
        ]);
    }
}
```

## Meta Boxes and Custom Fields

### Registering Meta for REST API

```php
public function registerMeta(): void
{
    register_post_meta('book', '_book_isbn', [
        'type'              => 'string',
        'single'            => true,
        'show_in_rest'      => true,
        'sanitize_callback' => 'sanitize_text_field',
        'auth_callback'     => fn() => current_user_can('edit_posts'),
    ]);

    register_post_meta('book', '_book_price', [
        'type'              => 'number',
        'single'            => true,
        'show_in_rest'      => true,
        'sanitize_callback' => fn($value) => floatval($value),
    ]);
}
```

### Meta Box with Save Handler

```php
<?php
declare(strict_types=1);

namespace MyPlugin\Admin;

final class BookMetaBox
{
    public function __construct()
    {
        add_action('add_meta_boxes', [$this, 'addMetaBox']);
        add_action('save_post_book', [$this, 'saveMetaBox'], 10, 2);
        add_action('init', [$this, 'registerMeta']);
    }

    public function addMetaBox(): void
    {
        add_meta_box(
            'book_details',
            __('Book Details', 'my-plugin'),
            [$this, 'renderMetaBox'],
            'book',
            'normal',
            'high'
        );
    }

    public function renderMetaBox(\WP_Post $post): void
    {
        wp_nonce_field('book_meta_box', 'book_meta_box_nonce');

        $isbn = get_post_meta($post->ID, '_book_isbn', true);
        $price = get_post_meta($post->ID, '_book_price', true);

        ?>
        <table class="form-table">
            <tr>
                <th><label for="book_isbn"><?php esc_html_e('ISBN', 'my-plugin'); ?></label></th>
                <td>
                    <input type="text" id="book_isbn" name="book_isbn"
                           value="<?php echo esc_attr($isbn); ?>" class="regular-text">
                </td>
            </tr>
            <tr>
                <th><label for="book_price"><?php esc_html_e('Price', 'my-plugin'); ?></label></th>
                <td>
                    <input type="number" id="book_price" name="book_price" step="0.01"
                           value="<?php echo esc_attr($price); ?>" class="small-text">
                </td>
            </tr>
        </table>
        <?php
    }

    public function saveMetaBox(int $postId, \WP_Post $post): void
    {
        // Verify nonce
        if (!isset($_POST['book_meta_box_nonce']) ||
            !wp_verify_nonce($_POST['book_meta_box_nonce'], 'book_meta_box')) {
            return;
        }

        // Check autosave
        if (defined('DOING_AUTOSAVE') && DOING_AUTOSAVE) {
            return;
        }

        // Check permissions
        if (!current_user_can('edit_post', $postId)) {
            return;
        }

        // Save meta fields
        if (isset($_POST['book_isbn'])) {
            update_post_meta($postId, '_book_isbn', sanitize_text_field($_POST['book_isbn']));
        }

        if (isset($_POST['book_price'])) {
            update_post_meta($postId, '_book_price', floatval($_POST['book_price']));
        }
    }
}
```

## Database Operations

### Custom Tables

```php
<?php
declare(strict_types=1);

namespace MyPlugin;

final class Database
{
    public static function createTables(): void
    {
        global $wpdb;

        $charsetCollate = $wpdb->get_charset_collate();
        $tableName = $wpdb->prefix . 'book_reviews';

        $sql = "CREATE TABLE {$tableName} (
            id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
            book_id bigint(20) unsigned NOT NULL,
            user_id bigint(20) unsigned NOT NULL,
            rating tinyint(1) unsigned NOT NULL,
            review text NOT NULL,
            status varchar(20) NOT NULL DEFAULT 'pending',
            created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            PRIMARY KEY (id),
            KEY book_id (book_id),
            KEY user_id (user_id),
            KEY status (status)
        ) {$charsetCollate};";

        require_once ABSPATH . 'wp-admin/includes/upgrade.php';
        dbDelta($sql);

        update_option('my_plugin_db_version', MY_PLUGIN_VERSION);
    }

    public static function dropTables(): void
    {
        global $wpdb;

        $tableName = $wpdb->prefix . 'book_reviews';
        $wpdb->query("DROP TABLE IF EXISTS {$tableName}");

        delete_option('my_plugin_db_version');
    }
}
```

### Repository Pattern

```php
<?php
declare(strict_types=1);

namespace MyPlugin\Repository;

final class BookReviewRepository
{
    private \wpdb $wpdb;
    private string $table;

    public function __construct()
    {
        global $wpdb;
        $this->wpdb = $wpdb;
        $this->table = $wpdb->prefix . 'book_reviews';
    }

    public function create(array $data): int|false
    {
        $result = $this->wpdb->insert(
            $this->table,
            [
                'book_id' => $data['book_id'],
                'user_id' => $data['user_id'],
                'rating'  => $data['rating'],
                'review'  => $data['review'],
                'status'  => $data['status'] ?? 'pending',
            ],
            ['%d', '%d', '%d', '%s', '%s']
        );

        return $result ? $this->wpdb->insert_id : false;
    }

    public function findById(int $id): ?object
    {
        $result = $this->wpdb->get_row(
            $this->wpdb->prepare(
                "SELECT * FROM {$this->table} WHERE id = %d",
                $id
            )
        );

        return $result ?: null;
    }

    public function findByBookId(int $bookId, string $status = 'approved'): array
    {
        return $this->wpdb->get_results(
            $this->wpdb->prepare(
                "SELECT * FROM {$this->table}
                 WHERE book_id = %d AND status = %s
                 ORDER BY created_at DESC",
                $bookId,
                $status
            )
        );
    }

    public function getAverageRating(int $bookId): ?float
    {
        $result = $this->wpdb->get_var(
            $this->wpdb->prepare(
                "SELECT AVG(rating) FROM {$this->table}
                 WHERE book_id = %d AND status = 'approved'",
                $bookId
            )
        );

        return $result !== null ? round((float) $result, 1) : null;
    }

    public function update(int $id, array $data): bool
    {
        $updateData = [];
        $format = [];

        if (isset($data['rating'])) {
            $updateData['rating'] = $data['rating'];
            $format[] = '%d';
        }

        if (isset($data['review'])) {
            $updateData['review'] = $data['review'];
            $format[] = '%s';
        }

        if (isset($data['status'])) {
            $updateData['status'] = $data['status'];
            $format[] = '%s';
        }

        if (empty($updateData)) {
            return false;
        }

        $result = $this->wpdb->update(
            $this->table,
            $updateData,
            ['id' => $id],
            $format,
            ['%d']
        );

        return $result !== false;
    }

    public function delete(int $id): bool
    {
        $result = $this->wpdb->delete(
            $this->table,
            ['id' => $id],
            ['%d']
        );

        return $result !== false;
    }
}
```

**Database conventions:**
- Always use `$wpdb->prepare()` for parameterized queries
- Use `dbDelta()` for table creation (supports upgrades)
- Prefix table names with `$wpdb->prefix`
- Track schema version with `update_option()`
- Use format specifiers: `%d` (integer), `%s` (string), `%f` (float)

## Gutenberg Block (Full Example)

### Dynamic Block with Server-Side Render

```php
<?php
declare(strict_types=1);

namespace MyPlugin\Blocks;

final class BlockManager
{
    public function __construct()
    {
        add_action('init', [$this, 'registerBlocks']);
    }

    public function registerBlocks(): void
    {
        // Static block from block.json
        register_block_type(MY_PLUGIN_PATH . 'blocks/book-card');

        // Dynamic block with PHP render
        register_block_type('my-plugin/featured-books', [
            'render_callback' => [$this, 'renderFeaturedBooks'],
            'attributes'      => [
                'count' => ['type' => 'number', 'default' => 3],
                'genre' => ['type' => 'string', 'default' => ''],
            ],
        ]);
    }

    public function renderFeaturedBooks(array $attributes): string
    {
        $args = [
            'post_type'      => 'book',
            'posts_per_page' => $attributes['count'],
            'orderby'        => 'date',
            'order'          => 'DESC',
        ];

        if (!empty($attributes['genre'])) {
            $args['tax_query'] = [
                [
                    'taxonomy' => 'genre',
                    'field'    => 'slug',
                    'terms'    => $attributes['genre'],
                ],
            ];
        }

        $query = new \WP_Query($args);

        if (!$query->have_posts()) {
            return '<p>' . esc_html__('No books found.', 'my-plugin') . '</p>';
        }

        ob_start();
        ?>
        <div class="wp-block-my-plugin-featured-books">
            <?php while ($query->have_posts()) : $query->the_post(); ?>
                <article class="book-card">
                    <?php if (has_post_thumbnail()) : ?>
                        <div class="book-card__image">
                            <?php the_post_thumbnail('medium'); ?>
                        </div>
                    <?php endif; ?>
                    <div class="book-card__content">
                        <h3 class="book-card__title">
                            <a href="<?php the_permalink(); ?>"><?php the_title(); ?></a>
                        </h3>
                        <?php the_excerpt(); ?>
                    </div>
                </article>
            <?php endwhile; ?>
        </div>
        <?php
        wp_reset_postdata();

        return ob_get_clean();
    }
}
```

### Block JavaScript (Full Edit Component)

```javascript
import { registerBlockType } from '@wordpress/blocks';
import { useBlockProps, InspectorControls } from '@wordpress/block-editor';
import { PanelBody, ToggleControl, ComboboxControl } from '@wordpress/components';
import { useSelect } from '@wordpress/data';
import { __ } from '@wordpress/i18n';
import ServerSideRender from '@wordpress/server-side-render';

registerBlockType('my-plugin/book-card', {
    edit: ({ attributes, setAttributes }) => {
        const { bookId, showImage, showExcerpt } = attributes;
        const blockProps = useBlockProps();

        // Fetch books for the selector
        const books = useSelect((select) => {
            return select('core').getEntityRecords('postType', 'book', {
                per_page: 100,
                _fields: ['id', 'title'],
            });
        }, []);

        const bookOptions = (books || []).map((book) => ({
            value: book.id,
            label: book.title.rendered,
        }));

        return (
            <>
                <InspectorControls>
                    <PanelBody title={__('Book Settings', 'my-plugin')}>
                        <ComboboxControl
                            label={__('Select Book', 'my-plugin')}
                            value={bookId}
                            options={bookOptions}
                            onChange={(value) => setAttributes({ bookId: parseInt(value, 10) })}
                        />
                        <ToggleControl
                            label={__('Show Featured Image', 'my-plugin')}
                            checked={showImage}
                            onChange={(value) => setAttributes({ showImage: value })}
                        />
                        <ToggleControl
                            label={__('Show Excerpt', 'my-plugin')}
                            checked={showExcerpt}
                            onChange={(value) => setAttributes({ showExcerpt: value })}
                        />
                    </PanelBody>
                </InspectorControls>
                <div {...blockProps}>
                    {bookId ? (
                        <ServerSideRender
                            block="my-plugin/book-card"
                            attributes={attributes}
                        />
                    ) : (
                        <p>{__('Please select a book.', 'my-plugin')}</p>
                    )}
                </div>
            </>
        );
    },
    save: () => null, // Dynamic block: rendered on server
});
```

**Block development conventions:**
- Use `block.json` for all block metadata (preferred over PHP registration)
- Use `apiVersion: 3` for latest block API
- Use `ServerSideRender` for dynamic blocks that need PHP rendering
- Use `InspectorControls` for sidebar settings
- Use `useBlockProps()` for proper block wrapper attributes
- Use `useSelect` from `@wordpress/data` to query WordPress data
- Always call `wp_reset_postdata()` after custom queries in render callbacks

## WooCommerce Integration

### Custom Product Data Tab

```php
<?php
declare(strict_types=1);

namespace MyPlugin\WooCommerce;

final class ProductTab
{
    public function __construct()
    {
        add_filter('woocommerce_product_data_tabs', [$this, 'addTab']);
        add_action('woocommerce_product_data_panels', [$this, 'addPanel']);
        add_action('woocommerce_process_product_meta', [$this, 'saveFields']);
    }

    public function addTab(array $tabs): array
    {
        $tabs['book_info'] = [
            'label'    => __('Book Info', 'my-plugin'),
            'target'   => 'book_info_panel',
            'class'    => [],
            'priority' => 80,
        ];

        return $tabs;
    }

    public function addPanel(): void
    {
        ?>
        <div id="book_info_panel" class="panel woocommerce_options_panel">
            <?php
            woocommerce_wp_text_input([
                'id'    => '_book_isbn',
                'label' => __('ISBN', 'my-plugin'),
            ]);

            woocommerce_wp_text_input([
                'id'          => '_book_pages',
                'label'       => __('Pages', 'my-plugin'),
                'type'        => 'number',
                'desc_tip'    => true,
                'description' => __('Number of pages in the book.', 'my-plugin'),
            ]);

            woocommerce_wp_select([
                'id'      => '_book_format',
                'label'   => __('Format', 'my-plugin'),
                'options' => [
                    ''          => __('Select format', 'my-plugin'),
                    'hardcover' => __('Hardcover', 'my-plugin'),
                    'paperback' => __('Paperback', 'my-plugin'),
                    'ebook'     => __('E-Book', 'my-plugin'),
                ],
            ]);
            ?>
        </div>
        <?php
    }

    public function saveFields(int $postId): void
    {
        if (isset($_POST['_book_isbn'])) {
            update_post_meta($postId, '_book_isbn', sanitize_text_field($_POST['_book_isbn']));
        }

        if (isset($_POST['_book_pages'])) {
            update_post_meta($postId, '_book_pages', absint($_POST['_book_pages']));
        }

        if (isset($_POST['_book_format'])) {
            update_post_meta($postId, '_book_format', sanitize_text_field($_POST['_book_format']));
        }
    }
}
```

### WooCommerce Hooks Reference

```php
// Order processing
add_action('woocommerce_order_status_completed', [$this, 'onOrderComplete']);
add_action('woocommerce_thankyou', [$this, 'onThankYou']);

// Cart
add_action('woocommerce_before_add_to_cart_button', [$this, 'beforeAddToCart']);
add_filter('woocommerce_add_to_cart_validation', [$this, 'validateAddToCart'], 10, 3);

// Checkout
add_action('woocommerce_checkout_process', [$this, 'validateCheckout']);
add_action('woocommerce_checkout_create_order', [$this, 'modifyOrder']);

// Product display
add_filter('woocommerce_product_tabs', [$this, 'addProductTab']);
add_action('woocommerce_single_product_summary', [$this, 'addProductInfo'], 25);

// Admin columns
add_filter('manage_edit-product_columns', [$this, 'addAdminColumn']);
add_action('manage_product_posts_custom_column', [$this, 'renderAdminColumn'], 10, 2);
```

## Testing

### PHPUnit Bootstrap

```php
<?php
// tests/bootstrap.php

$_tests_dir = getenv('WP_TESTS_DIR') ?: '/tmp/wordpress-tests-lib';

require_once $_tests_dir . '/includes/functions.php';

function _manually_load_plugin(): void
{
    require dirname(__DIR__) . '/my-plugin.php';
}

tests_add_filter('muplugins_loaded', '_manually_load_plugin');

require $_tests_dir . '/includes/bootstrap.php';
```

### Unit Test Example

```php
<?php
declare(strict_types=1);

namespace MyPlugin\Tests;

use MyPlugin\Repository\BookReviewRepository;
use WP_UnitTestCase;

final class BookReviewRepositoryTest extends WP_UnitTestCase
{
    private BookReviewRepository $repository;
    private int $bookId;
    private int $userId;

    public function setUp(): void
    {
        parent::setUp();

        $this->repository = new BookReviewRepository();
        $this->bookId = $this->factory->post->create(['post_type' => 'book']);
        $this->userId = $this->factory->user->create();
    }

    public function test_create_review_returns_positive_id(): void
    {
        $reviewId = $this->repository->create([
            'book_id' => $this->bookId,
            'user_id' => $this->userId,
            'rating'  => 5,
            'review'  => 'Great book!',
        ]);

        $this->assertIsInt($reviewId);
        $this->assertGreaterThan(0, $reviewId);
    }

    public function test_find_by_id_returns_correct_review(): void
    {
        $reviewId = $this->repository->create([
            'book_id' => $this->bookId,
            'user_id' => $this->userId,
            'rating'  => 4,
            'review'  => 'Good read.',
        ]);

        $review = $this->repository->findById($reviewId);

        $this->assertNotNull($review);
        $this->assertEquals($this->bookId, $review->book_id);
        $this->assertEquals(4, $review->rating);
    }

    public function test_get_average_rating_calculates_correctly(): void
    {
        $this->repository->create([
            'book_id' => $this->bookId,
            'user_id' => $this->userId,
            'rating'  => 5,
            'review'  => 'Excellent!',
            'status'  => 'approved',
        ]);

        $this->repository->create([
            'book_id' => $this->bookId,
            'user_id' => $this->factory->user->create(),
            'rating'  => 3,
            'review'  => 'Okay.',
            'status'  => 'approved',
        ]);

        $average = $this->repository->getAverageRating($this->bookId);

        $this->assertEquals(4.0, $average);
    }

    public function test_delete_review_removes_record(): void
    {
        $reviewId = $this->repository->create([
            'book_id' => $this->bookId,
            'user_id' => $this->userId,
            'rating'  => 4,
            'review'  => 'To be deleted.',
        ]);

        $deleted = $this->repository->delete($reviewId);

        $this->assertTrue($deleted);
        $this->assertNull($this->repository->findById($reviewId));
    }
}
```

**Testing conventions:**
- Extend `WP_UnitTestCase` for WordPress integration tests
- Use `$this->factory` to create test fixtures (posts, users, terms)
- Tests are isolated: each test runs in a database transaction that rolls back
- Name tests descriptively: `test_create_review_returns_positive_id`
- Test file: `tests/phpunit/BookReviewRepositoryTest.php`

### PHPUnit Configuration

```xml
<!-- phpunit.xml.dist -->
<phpunit bootstrap="tests/bootstrap.php" colors="true">
    <testsuites>
        <testsuite name="Plugin Tests">
            <directory>tests/phpunit</directory>
        </testsuite>
    </testsuites>
    <coverage>
        <include>
            <directory suffix=".php">includes</directory>
        </include>
    </coverage>
</phpunit>
```

## Performance and Caching

### Transient API (Database Cache)

```php
function get_featured_books(): array
{
    $cacheKey = 'my_plugin_featured_books';
    $books = get_transient($cacheKey);

    if ($books === false) {
        $books = get_posts([
            'post_type'      => 'book',
            'posts_per_page' => 10,
            'meta_key'       => '_featured',
            'meta_value'     => '1',
        ]);

        set_transient($cacheKey, $books, HOUR_IN_SECONDS);
    }

    return $books;
}

// Invalidate cache when data changes
add_action('save_post_book', function (int $postId): void {
    delete_transient('my_plugin_featured_books');
});
```

### Object Cache (In-Memory)

```php
function get_book_meta_cached(int $postId): array
{
    $cacheKey = "book_meta_{$postId}";
    $meta = wp_cache_get($cacheKey, 'my_plugin');

    if ($meta === false) {
        $meta = [
            'isbn'  => get_post_meta($postId, '_book_isbn', true),
            'price' => get_post_meta($postId, '_book_price', true),
        ];
        wp_cache_set($cacheKey, $meta, 'my_plugin', 3600);
    }

    return $meta;
}
```

### Query Optimization

```php
// Optimize WP_Query for performance
$books = new WP_Query([
    'post_type'              => 'book',
    'posts_per_page'         => 10,
    'no_found_rows'          => true,  // Skip pagination count query
    'update_post_meta_cache' => false, // Skip meta cache if not needed
    'update_post_term_cache' => false, // Skip term cache if not needed
    'fields'                 => 'ids', // Only get IDs when full objects unnecessary
]);
```

**Performance guidelines:**
- Use `get_transient()` / `set_transient()` for expensive queries with TTL
- Use `wp_cache_get()` / `wp_cache_set()` for request-scoped caching
- Always invalidate caches on relevant `save_post` / `updated_post_meta` hooks
- Use `no_found_rows => true` when pagination count is not needed
- Use `fields => 'ids'` when you only need post IDs
- Use `update_post_meta_cache => false` when post meta is not accessed
- WordPress time constants: `MINUTE_IN_SECONDS`, `HOUR_IN_SECONDS`, `DAY_IN_SECONDS`, `WEEK_IN_SECONDS`

## AJAX Handling

### PHP Handler

```php
public function __construct()
{
    // Authenticated users
    add_action('wp_ajax_my_action', [$this, 'handleAjax']);
    // Non-authenticated users (public)
    add_action('wp_ajax_nopriv_my_action', [$this, 'handleAjax']);
}

public function handleAjax(): void
{
    check_ajax_referer('my_plugin_nonce', 'nonce');

    $action = sanitize_text_field($_POST['custom_action'] ?? '');

    switch ($action) {
        case 'get_books':
            $books = get_posts([
                'post_type'      => 'book',
                'posts_per_page' => 10,
            ]);

            $data = array_map(fn($book) => [
                'id'    => $book->ID,
                'title' => $book->post_title,
                'url'   => get_permalink($book),
            ], $books);

            wp_send_json_success($data);
            break;
        default:
            wp_send_json_error(['message' => __('Invalid action.', 'my-plugin')]);
    }
}
```

### JavaScript AJAX Call

```javascript
jQuery(document).ready(function ($) {
    $.ajax({
        url: MyPluginData.ajaxUrl,
        type: 'POST',
        data: {
            action: 'my_action',
            custom_action: 'get_books',
            nonce: MyPluginData.nonce,
        },
        success: function (response) {
            if (response.success) {
                console.log(response.data);
            } else {
                console.error(response.data.message);
            }
        },
        error: function () {
            console.error('Request failed');
        },
    });
});
```

## Custom WP-CLI Commands

```php
<?php
declare(strict_types=1);

namespace MyPlugin\CLI;

use WP_CLI;
use WP_CLI_Command;

if (!defined('WP_CLI') || !WP_CLI) {
    return;
}

final class BookCommand extends WP_CLI_Command
{
    /**
     * List all books.
     *
     * ## OPTIONS
     *
     * [--format=<format>]
     * : Output format (table, json, csv).
     * ---
     * default: table
     * ---
     *
     * ## EXAMPLES
     *
     *     wp book list
     *     wp book list --format=json
     *
     * @when after_wp_load
     */
    public function list(array $args, array $assocArgs): void
    {
        $books = get_posts([
            'post_type'      => 'book',
            'posts_per_page' => -1,
            'post_status'    => $assocArgs['status'] ?? 'publish',
        ]);

        if (empty($books)) {
            WP_CLI::warning('No books found.');
            return;
        }

        $items = array_map(fn($book) => [
            'ID'     => $book->ID,
            'Title'  => $book->post_title,
            'Status' => $book->post_status,
            'ISBN'   => get_post_meta($book->ID, '_book_isbn', true),
        ], $books);

        WP_CLI\Utils\format_items(
            $assocArgs['format'] ?? 'table',
            $items,
            ['ID', 'Title', 'Status', 'ISBN']
        );
    }

    /**
     * Import books from CSV.
     *
     * ## OPTIONS
     *
     * <file>
     * : Path to CSV file.
     *
     * [--dry-run]
     * : Preview import without creating posts.
     *
     * ## EXAMPLES
     *
     *     wp book import books.csv
     *     wp book import books.csv --dry-run
     *
     * @when after_wp_load
     */
    public function import(array $args, array $assocArgs): void
    {
        $file = $args[0];
        $dryRun = isset($assocArgs['dry-run']);

        if (!file_exists($file)) {
            WP_CLI::error("File not found: {$file}");
        }

        $handle = fopen($file, 'r');
        $headers = fgetcsv($handle);
        $count = 0;

        while (($row = fgetcsv($handle)) !== false) {
            $data = array_combine($headers, $row);

            if ($dryRun) {
                WP_CLI::log("Would import: {$data['title']}");
            } else {
                $postId = wp_insert_post([
                    'post_type'   => 'book',
                    'post_title'  => $data['title'],
                    'post_status' => 'publish',
                ]);

                if (!is_wp_error($postId)) {
                    if (!empty($data['isbn'])) {
                        update_post_meta($postId, '_book_isbn', $data['isbn']);
                    }
                    $count++;
                }
            }
        }

        fclose($handle);

        WP_CLI::success($dryRun ? 'Dry run completed.' : "Imported {$count} books.");
    }
}

WP_CLI::add_command('book', BookCommand::class);
```

## Theme Patterns

### Theme Setup (functions.php)

```php
<?php
declare(strict_types=1);

namespace MyTheme;

add_action('after_setup_theme', function (): void {
    // Enable features
    add_theme_support('title-tag');
    add_theme_support('post-thumbnails');
    add_theme_support('automatic-feed-links');
    add_theme_support('html5', ['search-form', 'comment-form', 'comment-list', 'gallery', 'caption']);
    add_theme_support('customize-selective-refresh-widgets');
    add_theme_support('wp-block-styles');
    add_theme_support('responsive-embeds');
    add_theme_support('editor-styles');

    // Custom image sizes
    add_image_size('card-thumbnail', 400, 300, true);

    // Navigation menus
    register_nav_menus([
        'primary'  => __('Primary Menu', 'my-theme'),
        'footer'   => __('Footer Menu', 'my-theme'),
    ]);
});

// Enqueue theme assets
add_action('wp_enqueue_scripts', function (): void {
    $version = wp_get_theme()->get('Version');

    wp_enqueue_style('my-theme-style', get_stylesheet_uri(), [], $version);
    wp_enqueue_script('my-theme-script', get_template_directory_uri() . '/assets/js/main.js', [], $version, true);
});

// Widget areas
add_action('widgets_init', function (): void {
    register_sidebar([
        'name'          => __('Primary Sidebar', 'my-theme'),
        'id'            => 'sidebar-1',
        'before_widget' => '<section id="%1$s" class="widget %2$s">',
        'after_widget'  => '</section>',
        'before_title'  => '<h2 class="widget-title">',
        'after_title'   => '</h2>',
    ]);
});
```

### Full Site Editing (theme.json)

```json
{
    "$schema": "https://schemas.wp.org/trunk/theme.json",
    "version": 2,
    "settings": {
        "color": {
            "palette": [
                { "slug": "primary", "color": "#1e40af", "name": "Primary" },
                { "slug": "secondary", "color": "#7c3aed", "name": "Secondary" },
                { "slug": "background", "color": "#ffffff", "name": "Background" },
                { "slug": "foreground", "color": "#1f2937", "name": "Foreground" }
            ]
        },
        "typography": {
            "fontFamilies": [
                {
                    "fontFamily": "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
                    "slug": "system",
                    "name": "System"
                }
            ],
            "fontSizes": [
                { "slug": "small", "size": "0.875rem", "name": "Small" },
                { "slug": "medium", "size": "1rem", "name": "Medium" },
                { "slug": "large", "size": "1.5rem", "name": "Large" }
            ]
        },
        "spacing": {
            "units": ["px", "em", "rem", "%"]
        },
        "layout": {
            "contentSize": "800px",
            "wideSize": "1200px"
        }
    },
    "styles": {
        "color": {
            "background": "var(--wp--preset--color--background)",
            "text": "var(--wp--preset--color--foreground)"
        }
    }
}
```
