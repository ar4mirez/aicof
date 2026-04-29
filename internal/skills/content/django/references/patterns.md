# Django Patterns Reference

## Contents

- [Settings Configuration](#settings-configuration)
- [Models with Relationships](#models-with-relationships)
- [Soft Delete Pattern](#soft-delete-pattern)
- [Services (Business Logic)](#services-business-logic)
- [DRF Serializers](#drf-serializers)
- [DRF ViewSets](#drf-viewsets)
- [Forms](#forms)
- [URL Configuration with DRF Router](#url-configuration-with-drf-router)
- [Signals](#signals)
- [Middleware](#middleware)
- [Celery Tasks](#celery-tasks)
- [Management Commands](#management-commands)
- [Testing Patterns](#testing-patterns)
- [Deployment Settings](#deployment-settings)

## Settings Configuration

### Base Settings

```python
# config/settings/base.py
from pathlib import Path
from decouple import config

BASE_DIR = Path(__file__).resolve().parent.parent.parent

SECRET_KEY = config("SECRET_KEY")

INSTALLED_APPS = [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
    # Third-party
    "rest_framework",
    "corsheaders",
    "django_extensions",
    # Local apps
    "apps.users",
    "apps.core",
]

MIDDLEWARE = [
    "django.middleware.security.SecurityMiddleware",
    "whitenoise.middleware.WhiteNoiseMiddleware",
    "corsheaders.middleware.CorsMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
    "django.middleware.clickjacking.XFrameOptionsMiddleware",
]

ROOT_URLCONF = "config.urls"

TEMPLATES = [
    {
        "BACKEND": "django.template.backends.django.DjangoTemplates",
        "DIRS": [BASE_DIR / "templates"],
        "APP_DIRS": True,
        "OPTIONS": {
            "context_processors": [
                "django.template.context_processors.debug",
                "django.template.context_processors.request",
                "django.contrib.auth.context_processors.auth",
                "django.contrib.messages.context_processors.messages",
            ],
        },
    },
]

# Database
DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": config("DB_NAME"),
        "USER": config("DB_USER"),
        "PASSWORD": config("DB_PASSWORD"),
        "HOST": config("DB_HOST", default="localhost"),
        "PORT": config("DB_PORT", default="5432"),
    }
}

AUTH_USER_MODEL = "users.User"

STATIC_URL = "static/"
STATIC_ROOT = BASE_DIR / "staticfiles"
STATICFILES_DIRS = [BASE_DIR / "static"]

MEDIA_URL = "media/"
MEDIA_ROOT = BASE_DIR / "media"

REST_FRAMEWORK = {
    "DEFAULT_AUTHENTICATION_CLASSES": [
        "rest_framework.authentication.SessionAuthentication",
        "rest_framework_simplejwt.authentication.JWTAuthentication",
    ],
    "DEFAULT_PERMISSION_CLASSES": [
        "rest_framework.permissions.IsAuthenticated",
    ],
    "DEFAULT_PAGINATION_CLASS": (
        "rest_framework.pagination.PageNumberPagination"
    ),
    "PAGE_SIZE": 20,
    "DEFAULT_THROTTLE_RATES": {
        "anon": "100/hour",
        "user": "1000/hour",
    },
}
```

### Development Settings

```python
# config/settings/dev.py
from .base import *

DEBUG = True
ALLOWED_HOSTS = ["localhost", "127.0.0.1"]

INSTALLED_APPS += ["debug_toolbar"]
MIDDLEWARE.insert(0, "debug_toolbar.middleware.DebugToolbarMiddleware")
INTERNAL_IPS = ["127.0.0.1"]

EMAIL_BACKEND = "django.core.mail.backends.console.EmailBackend"
CORS_ALLOW_ALL_ORIGINS = True
```

## Models with Relationships

```python
# apps/products/models.py
from django.db import models
from django.core.validators import MinValueValidator
from decimal import Decimal

from apps.core.models import TimeStampedModel, UUIDModel


class Category(TimeStampedModel):
    name = models.CharField(max_length=100, unique=True)
    slug = models.SlugField(unique=True)
    parent = models.ForeignKey(
        "self",
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="children",
    )

    class Meta:
        db_table = "categories"
        verbose_name_plural = "categories"

    def __str__(self) -> str:
        return self.name


class Product(UUIDModel, TimeStampedModel):
    class Status(models.TextChoices):
        DRAFT = "draft", "Draft"
        PUBLISHED = "published", "Published"
        ARCHIVED = "archived", "Archived"

    name = models.CharField(max_length=255)
    slug = models.SlugField(unique=True)
    description = models.TextField()
    price = models.DecimalField(
        max_digits=10,
        decimal_places=2,
        validators=[MinValueValidator(Decimal("0.01"))],
    )
    category = models.ForeignKey(
        Category,
        on_delete=models.PROTECT,
        related_name="products",
    )
    status = models.CharField(
        max_length=20,
        choices=Status.choices,
        default=Status.DRAFT,
    )
    stock = models.PositiveIntegerField(default=0)

    class Meta:
        db_table = "products"
        ordering = ["-created_at"]
        indexes = [
            models.Index(fields=["slug"]),
            models.Index(fields=["status"]),
            models.Index(fields=["category", "status"]),
        ]

    def __str__(self) -> str:
        return self.name

    @property
    def is_available(self) -> bool:
        return self.status == self.Status.PUBLISHED and self.stock > 0
```

## Soft Delete Pattern

```python
# apps/core/models.py
class SoftDeleteModel(models.Model):
    """Abstract base model with soft delete."""
    is_deleted = models.BooleanField(default=False)
    deleted_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        abstract = True

    def delete(self, using=None, keep_parents=False):
        from django.utils import timezone
        self.is_deleted = True
        self.deleted_at = timezone.now()
        self.save(update_fields=["is_deleted", "deleted_at"])

    def hard_delete(self):
        super().delete()
```

## Services (Business Logic)

```python
# apps/products/services.py
from django.db import transaction
from django.core.exceptions import ValidationError
from typing import Optional

from .models import Product, Category


class ProductService:
    """Business logic for products -- keep views thin."""

    @staticmethod
    @transaction.atomic
    def create_product(
        *,
        name: str,
        description: str,
        price: float,
        category_id: int,
        stock: int = 0,
        created_by,
    ) -> Product:
        """Create a new product with validation."""
        try:
            category = Category.objects.get(id=category_id)
        except Category.DoesNotExist:
            raise ValidationError("Category not found")

        if Product.objects.filter(
            name=name, category=category
        ).exists():
            raise ValidationError(
                "Product with this name already exists in category"
            )

        return Product.objects.create(
            name=name,
            description=description,
            price=price,
            category=category,
            stock=stock,
            created_by=created_by,
        )

    @staticmethod
    def update_stock(product_id: int, quantity: int) -> Product:
        """Update product stock with optimistic locking."""
        with transaction.atomic():
            product = (
                Product.objects.select_for_update().get(id=product_id)
            )
            new_stock = product.stock + quantity

            if new_stock < 0:
                raise ValidationError("Insufficient stock")

            product.stock = new_stock
            product.save(update_fields=["stock", "updated_at"])
            return product

    @staticmethod
    def get_products_by_category(
        category_slug: str,
        limit: Optional[int] = None,
    ) -> list[Product]:
        """Get published products by category."""
        queryset = Product.objects.filter(
            category__slug=category_slug,
            status=Product.Status.PUBLISHED,
        ).select_related("category")

        if limit:
            queryset = queryset[:limit]

        return list(queryset)
```

## DRF Serializers

```python
# apps/products/serializers.py
from rest_framework import serializers
from django.utils.text import slugify

from .models import Product, Category


class CategorySerializer(serializers.ModelSerializer):
    class Meta:
        model = Category
        fields = ["id", "name", "slug"]


class ProductSerializer(serializers.ModelSerializer):
    """Read serializer with nested category."""
    category = CategorySerializer(read_only=True)
    is_available = serializers.BooleanField(read_only=True)

    class Meta:
        model = Product
        fields = [
            "id", "name", "slug", "description", "price",
            "category", "status", "stock", "is_available",
            "created_at", "updated_at",
        ]
        read_only_fields = [
            "id", "slug", "created_at", "updated_at",
        ]


class ProductCreateSerializer(serializers.ModelSerializer):
    """Write serializer with flat category_id."""
    category_id = serializers.UUIDField(write_only=True)

    class Meta:
        model = Product
        fields = [
            "name", "description", "price",
            "category_id", "stock",
        ]

    def validate_category_id(self, value):
        if not Category.objects.filter(id=value).exists():
            raise serializers.ValidationError("Category not found")
        return value

    def validate_price(self, value):
        if value <= 0:
            raise serializers.ValidationError(
                "Price must be positive"
            )
        return value

    def create(self, validated_data):
        category_id = validated_data.pop("category_id")
        validated_data["category_id"] = category_id
        validated_data["slug"] = slugify(validated_data["name"])
        return super().create(validated_data)
```

## DRF ViewSets

```python
# apps/products/views_api.py
from rest_framework import viewsets, filters
from rest_framework.decorators import action
from rest_framework.response import Response
from rest_framework.permissions import IsAuthenticatedOrReadOnly
from django_filters.rest_framework import DjangoFilterBackend

from .models import Product
from .serializers import ProductSerializer, ProductCreateSerializer
from .filters import ProductFilter


class ProductViewSet(viewsets.ModelViewSet):
    """
    list:    Get all published products
    retrieve: Get single product by ID
    create:  Create new product (authenticated)
    update:  Update product (owner only)
    destroy: Delete product (owner only)
    """
    queryset = Product.objects.select_related(
        "category"
    ).prefetch_related("images")
    permission_classes = [IsAuthenticatedOrReadOnly]
    filter_backends = [
        DjangoFilterBackend,
        filters.SearchFilter,
        filters.OrderingFilter,
    ]
    filterset_class = ProductFilter
    search_fields = ["name", "description"]
    ordering_fields = ["price", "created_at", "name"]
    ordering = ["-created_at"]

    def get_serializer_class(self):
        if self.action in ["create", "update", "partial_update"]:
            return ProductCreateSerializer
        return ProductSerializer

    def get_queryset(self):
        queryset = super().get_queryset()
        if self.action == "list":
            queryset = queryset.filter(
                status=Product.Status.PUBLISHED
            )
        return queryset

    def perform_create(self, serializer):
        serializer.save(created_by=self.request.user)

    @action(detail=True, methods=["post"])
    def publish(self, request, pk=None):
        product = self.get_object()
        product.status = Product.Status.PUBLISHED
        product.save(update_fields=["status"])
        return Response({"status": "published"})

    @action(detail=False, methods=["get"])
    def featured(self, request):
        featured = self.get_queryset().filter(
            is_featured=True
        )[:10]
        serializer = self.get_serializer(featured, many=True)
        return Response(serializer.data)
```

## Forms

```python
# apps/products/forms.py
from django import forms
from django.core.exceptions import ValidationError
from django.utils.text import slugify

from .models import Product


class ProductForm(forms.ModelForm):
    class Meta:
        model = Product
        fields = [
            "name", "description", "price", "category", "stock",
        ]
        widgets = {
            "description": forms.Textarea(attrs={"rows": 4}),
            "price": forms.NumberInput(
                attrs={"step": "0.01", "min": "0"}
            ),
        }

    def clean_name(self):
        name = self.cleaned_data.get("name")
        slug = slugify(name)

        qs = Product.objects.filter(slug=slug)
        if self.instance.pk:
            qs = qs.exclude(pk=self.instance.pk)

        if qs.exists():
            raise ValidationError(
                "A product with this name already exists"
            )
        return name

    def save(self, commit=True):
        instance = super().save(commit=False)
        instance.slug = slugify(instance.name)
        if commit:
            instance.save()
        return instance
```

## URL Configuration with DRF Router

```python
# config/urls.py
from django.contrib import admin
from django.urls import path, include
from django.conf import settings
from django.conf.urls.static import static

urlpatterns = [
    path("admin/", admin.site.urls),
    path("api/", include("apps.api.urls")),
    path("", include("apps.products.urls")),
    path("users/", include("apps.users.urls")),
]

if settings.DEBUG:
    urlpatterns += static(
        settings.MEDIA_URL, document_root=settings.MEDIA_ROOT
    )
    urlpatterns += [
        path("__debug__/", include("debug_toolbar.urls")),
    ]


# apps/api/urls.py
from django.urls import path, include
from rest_framework.routers import DefaultRouter
from rest_framework_simplejwt.views import (
    TokenObtainPairView,
    TokenRefreshView,
)

from apps.products.views_api import ProductViewSet
from apps.users.views_api import UserViewSet

router = DefaultRouter()
router.register(r"products", ProductViewSet)
router.register(r"users", UserViewSet)

urlpatterns = [
    path("", include(router.urls)),
    path(
        "token/",
        TokenObtainPairView.as_view(),
        name="token_obtain_pair",
    ),
    path(
        "token/refresh/",
        TokenRefreshView.as_view(),
        name="token_refresh",
    ),
]
```

## Signals

```python
# apps/products/signals.py
from django.db.models.signals import post_save, pre_delete
from django.dispatch import receiver
from django.core.cache import cache

from .models import Product


@receiver(post_save, sender=Product)
def invalidate_product_cache(sender, instance, **kwargs):
    """Invalidate cache when product is saved."""
    cache.delete(f"product_{instance.id}")
    cache.delete("featured_products")


@receiver(pre_delete, sender=Product)
def cleanup_product_files(sender, instance, **kwargs):
    """Clean up associated files before deletion."""
    for image in instance.images.all():
        image.file.delete(save=False)


# apps/products/apps.py
from django.apps import AppConfig


class ProductsConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "apps.products"

    def ready(self):
        import apps.products.signals  # noqa
```

**Note**: Use signals sparingly. Prefer explicit service calls for most use cases.
Signals are appropriate for cache invalidation and file cleanup. Avoid using them
for core business logic since they make debugging harder.

## Middleware

```python
# apps/core/middleware.py
import time
import logging

logger = logging.getLogger(__name__)


class RequestTimingMiddleware:
    """Log request timing for performance monitoring."""

    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        start_time = time.time()
        response = self.get_response(request)
        duration = time.time() - start_time

        logger.info(
            "%s %s - %s - %.3fs",
            request.method,
            request.path,
            response.status_code,
            duration,
        )
        return response
```

## Celery Tasks

```python
# apps/products/tasks.py
from celery import shared_task
from django.core.mail import send_mail
from django.conf import settings

from .models import Product


@shared_task
def send_low_stock_alert(product_id: int):
    """Send alert when product stock is low."""
    product = Product.objects.get(id=product_id)

    if product.stock <= 5:
        send_mail(
            subject=f"Low stock alert: {product.name}",
            message=(
                f"Product {product.name} has only "
                f"{product.stock} items left."
            ),
            from_email=settings.DEFAULT_FROM_EMAIL,
            recipient_list=[settings.ADMIN_EMAIL],
        )


@shared_task
def update_product_statistics():
    """Daily task to update product statistics."""
    from django.db.models import Count, Avg

    products = Product.objects.annotate(
        review_count=Count("reviews"),
        avg_rating=Avg("reviews__rating"),
    )

    for product in products:
        product.review_count_cached = product.review_count
        product.avg_rating_cached = product.avg_rating or 0

    Product.objects.bulk_update(
        products,
        ["review_count_cached", "avg_rating_cached"],
    )
```

## Management Commands

```python
# apps/products/management/commands/import_products.py
from django.core.management.base import BaseCommand, CommandError
import csv
from decimal import Decimal

from apps.products.models import Product, Category


class Command(BaseCommand):
    help = "Import products from CSV file"

    def add_arguments(self, parser):
        parser.add_argument(
            "csv_file", type=str, help="Path to CSV file"
        )
        parser.add_argument(
            "--dry-run",
            action="store_true",
            help="Preview import without saving",
        )

    def handle(self, *args, **options):
        csv_file = options["csv_file"]
        dry_run = options["dry_run"]

        try:
            with open(csv_file, "r") as f:
                reader = csv.DictReader(f)
                products = []

                for row in reader:
                    category = Category.objects.get(
                        slug=row["category_slug"]
                    )
                    product = Product(
                        name=row["name"],
                        slug=row["slug"],
                        description=row["description"],
                        price=Decimal(row["price"]),
                        category=category,
                        stock=int(row["stock"]),
                    )
                    products.append(product)

                if dry_run:
                    self.stdout.write(
                        f"Would import {len(products)} products"
                    )
                else:
                    Product.objects.bulk_create(products)
                    self.stdout.write(
                        self.style.SUCCESS(
                            f"Imported {len(products)} products"
                        )
                    )

        except FileNotFoundError:
            raise CommandError(f"File not found: {csv_file}")
        except Category.DoesNotExist:
            raise CommandError(
                f"Category not found: {row['category_slug']}"
            )
```

## Testing Patterns

### Model Tests

```python
# apps/products/tests/test_models.py
import pytest
from decimal import Decimal

from apps.products.models import Product, Category


@pytest.mark.django_db
class TestProductModel:
    def test_create_product(self, category):
        product = Product.objects.create(
            name="Test Product",
            slug="test-product",
            description="Test description",
            price=Decimal("19.99"),
            category=category,
            stock=10,
        )

        assert product.name == "Test Product"
        assert product.price == Decimal("19.99")
        assert product.is_available is False  # Draft status

    def test_product_is_available_when_published_with_stock(
        self, category
    ):
        product = Product.objects.create(
            name="Available Product",
            slug="available-product",
            description="Test",
            price=Decimal("10.00"),
            category=category,
            status=Product.Status.PUBLISHED,
            stock=5,
        )
        assert product.is_available is True

    def test_product_not_available_when_out_of_stock(
        self, category
    ):
        product = Product.objects.create(
            name="Out of Stock",
            slug="out-of-stock",
            description="Test",
            price=Decimal("10.00"),
            category=category,
            status=Product.Status.PUBLISHED,
            stock=0,
        )
        assert product.is_available is False
```

### API Tests

```python
# apps/products/tests/test_views.py
import pytest
from django.urls import reverse
from rest_framework import status

from apps.products.models import Product


@pytest.mark.django_db
class TestProductAPI:
    def test_list_products(
        self, api_client, published_products
    ):
        url = reverse("product-list")
        response = api_client.get(url)

        assert response.status_code == status.HTTP_200_OK
        assert (
            len(response.data["results"])
            == len(published_products)
        )

    def test_create_product_authenticated(
        self, authenticated_client, category
    ):
        url = reverse("product-list")
        data = {
            "name": "New Product",
            "description": "New description",
            "price": "29.99",
            "category_id": str(category.id),
            "stock": 10,
        }

        response = authenticated_client.post(url, data)

        assert (
            response.status_code == status.HTTP_201_CREATED
        )
        assert Product.objects.filter(
            name="New Product"
        ).exists()

    def test_create_product_unauthenticated(
        self, api_client, category
    ):
        url = reverse("product-list")
        data = {"name": "New Product"}

        response = api_client.post(url, data)

        assert (
            response.status_code
            == status.HTTP_401_UNAUTHORIZED
        )
```

### Test Fixtures

```python
# conftest.py
import pytest
from rest_framework.test import APIClient
from decimal import Decimal

from apps.users.models import User
from apps.products.models import Product, Category


@pytest.fixture
def api_client():
    return APIClient()


@pytest.fixture
def user(db):
    return User.objects.create_user(
        username="testuser",
        email="test@example.com",
        password="testpass123",
    )


@pytest.fixture
def authenticated_client(api_client, user):
    api_client.force_authenticate(user=user)
    return api_client


@pytest.fixture
def category(db):
    return Category.objects.create(
        name="Electronics", slug="electronics"
    )


@pytest.fixture
def published_products(db, category):
    products = []
    for i in range(5):
        products.append(
            Product.objects.create(
                name=f"Product {i}",
                slug=f"product-{i}",
                description="Test",
                price=Decimal("10.00"),
                category=category,
                status=Product.Status.PUBLISHED,
                stock=10,
            )
        )
    return products
```

## Deployment Settings

### Production Settings Checklist

```python
# config/settings/prod.py
from .base import *

DEBUG = False
ALLOWED_HOSTS = config("ALLOWED_HOSTS", cast=Csv())

# Security
SECURE_SSL_REDIRECT = True
SECURE_HSTS_SECONDS = 31536000
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_HSTS_PRELOAD = True
SESSION_COOKIE_SECURE = True
CSRF_COOKIE_SECURE = True
SECURE_BROWSER_XSS_FILTER = True
SECURE_CONTENT_TYPE_NOSNIFF = True
X_FRAME_OPTIONS = "DENY"

# Static files with WhiteNoise
STATICFILES_STORAGE = (
    "whitenoise.storage.CompressedManifestStaticFilesStorage"
)

# Logging
LOGGING = {
    "version": 1,
    "disable_existing_loggers": False,
    "formatters": {
        "verbose": {
            "format": (
                "{levelname} {asctime} {module} "
                "{process:d} {thread:d} {message}"
            ),
            "style": "{",
        },
    },
    "handlers": {
        "console": {
            "class": "logging.StreamHandler",
            "formatter": "verbose",
        },
    },
    "root": {
        "handlers": ["console"],
        "level": "WARNING",
    },
    "loggers": {
        "django": {
            "handlers": ["console"],
            "level": config("DJANGO_LOG_LEVEL", default="INFO"),
            "propagate": False,
        },
    },
}

# CORS -- explicit origins only
CORS_ALLOWED_ORIGINS = config(
    "CORS_ALLOWED_ORIGINS", cast=Csv()
)
```

### Docker Deployment

```dockerfile
# Dockerfile
FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

RUN apt-get update && apt-get install -y \
    libpq-dev gcc && \
    rm -rf /var/lib/apt/lists/*

COPY requirements/prod.txt requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
RUN python manage.py collectstatic --noinput

EXPOSE 8000

CMD ["gunicorn", "config.wsgi:application", \
     "--bind", "0.0.0.0:8000", \
     "--workers", "4", \
     "--timeout", "120"]
```
